package main

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waste3d/ghost-tunnel/api"
	tunnelgrpc "github.com/waste3d/ghost-tunnel/internal/interfaces/grpc"
	"google.golang.org/grpc"
)

func main() {

	connStr := "postgres://postgres:postgres@localhost:5432/ghost_tunnel?sslmode=disable"

	ctx := context.Background()

	dbpool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer dbpool.Close()

	if err := dbpool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping pool: %v", err)
	}

	sessionManager := tunnelgrpc.NewSessionManager()
	connMgr := tunnelgrpc.NewConnectionManager()

	go startGrpcServer(sessionManager, connMgr)
	startPublicServer(sessionManager, connMgr)
}

func startGrpcServer(sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager) {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	tunnelSrv := tunnelgrpc.NewTunnelServer(sm, connMgr)
	api.RegisterTunnelServiceServer(grpcServer, tunnelSrv)

	log.Printf("server listening at %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC: failed to serve: %v", err)
	}
}

func startPublicServer(sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager) {
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
		go handlePublicConnection(conn, sm, connMgr)
	}
}

func handlePublicConnection(conn net.Conn, sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager) {
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

	dataChan := make(chan []byte)
	connMgr.Add(connID, dataChan)

	go func() {
		for data := range dataChan {
			if _, err := conn.Write(data); err != nil {
				log.Printf("Error writing data to public connection: %v", err)
				return
			}
		}
	}()

	buf := make([]byte, 4*1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from public conn %s: %v", connID, err)
			}
			return // Выходим, когда соединение закрыто
		}

		sendErr := grpcStream.Send(&api.ServerToClient{
			Message: &api.ServerToClient_Data{
				Data: &api.Data{
					ConnectionId: connID,
					Chunk:        buf[:n],
				},
			},
		})
		if sendErr != nil {
			log.Printf("Error sending data to client for conn %s: %v", connID, sendErr)
			return
		}
	}
}
