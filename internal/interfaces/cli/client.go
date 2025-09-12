package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"github.com/waste3d/ghost-tunnel/api"
	"google.golang.org/grpc"
)

type connectionManager struct {
	connections map[string]chan []byte
	mu          sync.RWMutex
}

func newConnectionManager() *connectionManager {
	return &connectionManager{
		connections: make(map[string]chan []byte),
	}
}

type Client struct {
	grpcConn  *grpc.ClientConn
	stream    api.TunnelService_EstablishTunnelClient
	tunnelID  string
	localAddr string
	connMgr   *connectionManager
}

func NewClient(tunnelID, localAddr string) *Client {
	return &Client{
		tunnelID:  tunnelID,
		localAddr: localAddr,
		connMgr:   newConnectionManager(),
	}
}

func (c *Client) Run(ctx context.Context, serverAddr string) error {
	log.Printf("Connecting to server at %s...", serverAddr)

	conn, err := grpc.DialContext(ctx, serverAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	c.grpcConn = conn
	defer c.grpcConn.Close()

	log.Println("Connection established.")

	grpcClient := api.NewTunnelServiceClient(conn)

	stream, err := grpcClient.EstablishTunnel(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish tunnel: %v", err)
	}
	c.stream = stream

	log.Printf("Registering tunnel ID: %s", c.tunnelID)

	err = c.stream.Send(&api.ClientToServer{
		Message: &api.ClientToServer_Register{
			Register: &api.Register{
				TunnelId: c.tunnelID,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to send register message: %v", err)
	}

	go c.listenServer()

	<-c.stream.Context().Done()
	return c.stream.Context().Err()
}

func (c *Client) listenServer() {
	for {
		msg, err := c.stream.Recv()
		if err == io.EOF {
			log.Println("Server closed the connection.")
			return
		}

		if err != nil {
			log.Printf("Error receiving from server: %v", err)
			return
		}

		if newConn := msg.GetNewConnection(); newConn != nil {
			log.Printf("Received request for new connection: %s", newConn.GetConnectionId())
			go c.handleConnection(newConn.GetConnectionId())
		}

		if data := msg.GetData(); data != nil {
			c.connMgr.mu.RLock()
			ch, ok := c.connMgr.connections[data.GetConnectionId()]
			c.connMgr.mu.RUnlock()
			if ok {
				ch <- data.GetChunk()
			} else {
				log.Printf("Received data for unknown connection ID: %s", data.GetConnectionId())
			}
		}
	}
}

func (c *Client) handleConnection(connectionID string) {
	localConn, err := net.Dial("tcp", c.localAddr)
	if err != nil {
		log.Printf("Failed to connect to local address %s: %v", c.localAddr, err)
		return
	}
	defer localConn.Close()
	log.Printf("Connection %s: established to local service %s", connectionID, c.localAddr)

	dataChan := make(chan []byte)
	c.connMgr.mu.Lock()
	c.connMgr.connections[connectionID] = dataChan
	c.connMgr.mu.Unlock()

	defer func() {
		c.connMgr.mu.Lock()
		close(c.connMgr.connections[connectionID])
		delete(c.connMgr.connections, connectionID)
		c.connMgr.mu.Unlock()
		log.Printf("Connection %s: cleaned up.", connectionID)
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer localConn.Close()

		for data := range dataChan {
			if _, err := localConn.Write(data); err != nil {
				log.Printf("Connection %s: error writing to local socket: %v", connectionID, err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		buf := make([]byte, 4*1024)
		for {
			n, err := localConn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Connection %s: error reading from local socket: %v", connectionID, err)
				}
				return
			}

			sendErr := c.stream.Send(&api.ClientToServer{
				Message: &api.ClientToServer_Data{
					Data: &api.Data{
						ConnectionId: connectionID,
						Chunk:        buf[:n],
					},
				},
			})
			if sendErr != nil {
				log.Printf("Connection %s: error sending data to server: %v", connectionID, sendErr)
				return
			}
		}
	}()

	wg.Wait()
}
