package cli

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/waste3d/ghost-tunnel/api"
	"google.golang.org/grpc"
)

type Client struct {
	grpcConn  *grpc.ClientConn
	stream    api.TunnelService_EstablishTunnelClient
	tunnelID  string
	localAddr string
}

func NewClient(tunnelID, localAddr string) *Client {
	return &Client{
		tunnelID:  tunnelID,
		localAddr: localAddr,
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
		// TODO: Обработать входящие данные (msg.GetData()) и перенаправить их
		// в нужный локальный TCP сокет. Мы сделаем это в handleConnection.
	}
}

func (c *Client) handleConnection(connectionID string) {
	localConn, err := net.Dial("tcp", c.localAddr)
	if err != nil {
		log.Printf("Failed to connect to local service at %s: %v", c.localAddr, err)
		// TODO: сообщить серверу о неудачном соединении
		return
	}
	defer localConn.Close()
	log.Printf("Connection %s: established to local service %s", connectionID, c.localAddr)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := localConn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Connection %s: error reading from local socket: %v", connectionID, err)
				}
				break
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
				break
			}
		}
	}()

	// 3. В этой горутине мы будем читать данные от сервера и писать их
	// в локальный сокет. (Этот код нужно будет доработать, когда listenServer
	// начнет распределять Data сообщения). Пока это заглушка.

	// TODO: Ждем пока соединение не закроется

	<-c.stream.Context().Done()
	log.Printf("Connection %s: closed", connectionID)
}
