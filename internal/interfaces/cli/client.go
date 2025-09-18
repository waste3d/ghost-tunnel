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
	"google.golang.org/grpc/credentials/insecure"
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
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()
	c.grpcConn = conn

	log.Println("Connection established.")
	grpcClient := api.NewTunnelServiceClient(c.grpcConn)
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
		if err != nil {
			if err != io.EOF {
				log.Printf("Error receiving from server: %v", err)
			}
			c.connMgr.mu.Lock()
			for _, ch := range c.connMgr.connections {
				close(ch)
			}
			c.connMgr.connections = make(map[string]chan []byte)
			c.connMgr.mu.Unlock()
			return
		}
		if newConn := msg.GetNewConnection(); newConn != nil {
			connID := newConn.GetConnectionId()
			log.Printf("Received request for new connection: %s", connID)
			dataChan := make(chan []byte, 100)
			c.connMgr.mu.Lock()
			c.connMgr.connections[connID] = dataChan
			c.connMgr.mu.Unlock()
			go c.handleConnection(connID, dataChan)
		}
		if data := msg.GetData(); data != nil {
			c.connMgr.mu.RLock()
			ch, ok := c.connMgr.connections[data.GetConnectionId()]
			c.connMgr.mu.RUnlock()
			if ok {
				ch <- data.GetChunk()
			}
		}
	}
}

func (c *Client) handleConnection(connectionID string, dataChan chan []byte) {
	defer func() {
		c.connMgr.mu.Lock()
		delete(c.connMgr.connections, connectionID)
		c.connMgr.mu.Unlock()
		log.Printf("Connection %s: cleaned up.", connectionID)
	}()

	localConn, err := net.Dial("tcp", c.localAddr)
	if err != nil {
		log.Printf("Failed to connect to local service at %s: %v", c.localAddr, err)
		_ = c.stream.Send(&api.ClientToServer{
			Message: &api.ClientToServer_Data{
				Data: &api.Data{ConnectionId: connectionID, Chunk: nil},
			},
		})
		return
	}
	defer localConn.Close()
	log.Printf("Connection %s: established to local service %s", connectionID, c.localAddr)

	grpcWriter := &StreamWriter{stream: c.stream, connID: connectionID}

	go func() {
		io.Copy(grpcWriter, localConn)
		_ = c.stream.Send(&api.ClientToServer{
			Message: &api.ClientToServer_Data{
				Data: &api.Data{ConnectionId: connectionID, Chunk: nil},
			},
		})
	}()

	for data := range dataChan {
		if data == nil {
			return
		}
		if _, err := localConn.Write(data); err != nil {
			return
		}
	}
}

type StreamWriter struct {
	stream api.TunnelService_EstablishTunnelClient
	connID string
}

func (w *StreamWriter) Write(p []byte) (n int, err error) {
	err = w.stream.Send(&api.ClientToServer{
		Message: &api.ClientToServer_Data{
			Data: &api.Data{
				ConnectionId: w.connID,
				Chunk:        p,
			},
		},
	})
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
