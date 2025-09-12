package main

import (
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	"github.com/waste3d/ghost-tunnel/api"
	tunnelgrpc "github.com/waste3d/ghost-tunnel/internal/interfaces/grpc"
	"google.golang.org/grpc"
)

func main() {
	sessionManager := tunnelgrpc.NewSessionManager()

	go startGrpcServer(sessionManager)
	startPublicServer(sessionManager)
}

func startGrpcServer(sm *tunnelgrpc.SessionManager) {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	tunnelSrv := tunnelgrpc.NewTunnelServer(sm)
	api.RegisterTunnelServiceServer(grpcServer, tunnelSrv)

	log.Printf("server listening at %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC: failed to serve: %v", err)
	}
}

func startPublicServer(sm *tunnelgrpc.SessionManager) {
	publicPort := ":8000"
	lis, err := net.Listen("tcp", publicPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Public server listening on %s", publicPort)

	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Printf("Public server: failed to accept connection: %v", err)
			continue
		}
		go handlePublicConnection(conn, sm)
	}
}

func handlePublicConnection(conn net.Conn, sm *tunnelgrpc.SessionManager) {
	defer conn.Close()
	log.Printf("New public connection from %s", conn.RemoteAddr())

	// заглушка - в реальном будем получить из поддомена
	const targetTunnelID = "some-test-id"

	grpcStream, ok := sm.Get(targetTunnelID)
	if !ok {
		log.Printf("Failed to get gRPC stream for tunnel '%s'", targetTunnelID)
		return
	}

	connID := uuid.New().String()

	log.Printf("Sending NewConnection command (connID: %s) to client for tunnel '%s'", connID, targetTunnelID)

	err := grpcStream.Send(&api.ServerToClient{
		Message: &api.ServerToClient_NewConnection{
			NewConnection: &api.NewConnection{
				ConnectionId: connID,
			},
		},
	})

	if err != nil {
		log.Printf("Failed to send NewConnection to client: %v", err)
		return
	}

	// Начинаем проксировать данные.
	// TODO: Нам нужен способ получать Data-сообщения от клиента,
	// которые относятся именно к этому connID. Пока просто читаем из publicConn
	// и выбрасываем данные, чтобы соединение не висело.
	log.Printf("Proxying data for connID: %s", connID)

	bytesCopied, err := io.Copy(io.Discard, conn)
	if err != nil {
		log.Printf("Error during proxying for connID %s: %v", connID, err)
	}
	log.Printf("Copied %d bytes for connID %s", bytesCopied, connID)
}
