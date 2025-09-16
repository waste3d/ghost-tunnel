package main

import (
	"bufio"
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waste3d/ghost-tunnel/api"
	"github.com/waste3d/ghost-tunnel/internal/application"
	"github.com/waste3d/ghost-tunnel/internal/domain"
	"github.com/waste3d/ghost-tunnel/internal/infrastructure/persistence"
	tunnelgrpc "github.com/waste3d/ghost-tunnel/internal/interfaces/grpc"
	http_handlers "github.com/waste3d/ghost-tunnel/internal/interfaces/http"
	"google.golang.org/grpc"
)

func main() {
	connStr := "postgres://postgres:postgres@localhost:5432/ghost_tunnel?sslmode=disable"
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("Unable to parse connection string: %v\n", err)
	}
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "uuid",
			OID:   pgtype.UUIDOID,
			Codec: &pgtype.UUIDCodec{},
		})
		return nil
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create connection pool: %v\n", err)
	}
	defer dbpool.Close()

	if err = dbpool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database using pgxpool.")

	sessionManager := tunnelgrpc.NewSessionManager()
	connManager := tunnelgrpc.NewConnectionManager()

	tunnelRepo := persistence.NewPostgresTunnelRepository(dbpool)
	tunnelService := application.NewTunnelService(tunnelRepo)
	tunnelHandler := http_handlers.NewTunnelHandler(tunnelService)

	go startGrpcServer(sessionManager, connManager)
	go startPublicServer(sessionManager, connManager, tunnelRepo)
	startApiServer(tunnelHandler)
}

func startApiServer(tunnelHandler *http_handlers.TunnelHandler) {
	router := gin.Default()
	tunnelHandler.RegisterRoutes(router)
	port := ":8081"
	log.Printf("API server listening on %s", port)
	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to run API server: %v", err)
	}
}

func startGrpcServer(sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager) {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("gRPC: failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	tunnelSrv := tunnelgrpc.NewTunnelServer(sm, connMgr)
	api.RegisterTunnelServiceServer(grpcServer, tunnelSrv)
	log.Printf("gRPC server listening on %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC: failed to serve: %v", err)
	}
}

func startPublicServer(sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager, tunnelRepo domain.TunnelRepository) {
	publicPort := ":8000"
	lis, err := net.Listen("tcp", publicPort)
	if err != nil {
		log.Fatalf("Public server: failed to listen: %v", err)
	}
	defer lis.Close()
	log.Printf("Public server listening on %s", publicPort)
	for {
		conn, err := lis.Accept()
		if err != nil {
			log.Printf("Public server: failed to accept connection: %v", err)
			continue
		}
		go handlePublicConnection(conn, sm, connMgr, tunnelRepo)
	}
}

func handlePublicConnection(publicConn net.Conn, sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager, tunnelRepo domain.TunnelRepository) {
	defer publicConn.Close()

	reader := bufio.NewReader(publicConn)
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Printf("Could not parse incoming request as HTTP: %v", err)
		return
	}

	subdomain := strings.Split(req.Host, ".")[0]
	log.Printf("New public connection for subdomain '%s' from %s", subdomain, publicConn.RemoteAddr())

	tunnel, err := tunnelRepo.FindBySubdomain(context.Background(), subdomain)
	if err != nil || tunnel == nil {
		log.Printf("No tunnel found for subdomain '%s'.", subdomain)
		return
	}

	grpcStream, ok := sm.Get(string(tunnel.ID))
	if !ok {
		log.Printf("Tunnel for subdomain '%s' exists, but no client connected.", subdomain)
		return
	}

	connID := uuid.New().String()
	err = grpcStream.Send(&api.ServerToClient{
		Message: &api.ServerToClient_NewConnection{
			NewConnection: &api.NewConnection{ConnectionId: connID},
		},
	})
	if err != nil {
		return
	}

	dataChan := make(chan []byte, 100)
	connMgr.Add(connID, dataChan)
	defer connMgr.Remove(connID)

	log.Printf("Connection %s: starting bidirectional proxy for subdomain '%s'", connID, subdomain)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer publicConn.Close()
		for data := range dataChan {
			if _, err := publicConn.Write(data); err != nil {
				return
			}
		}
	}()

	go func() {
		defer wg.Done()

		rawRequest, err := httputil.DumpRequest(req, true)
		if err != nil {
			close(dataChan)
			return
		}

		err = grpcStream.Send(&api.ServerToClient{
			Message: &api.ServerToClient_Data{
				Data: &api.Data{ConnectionId: connID, Chunk: rawRequest},
			},
		})
		if err != nil {
			close(dataChan)
			return
		}

		buf := make([]byte, 4*1024)
		for {
			n, err := publicConn.Read(buf)
			if err != nil {
				close(dataChan)
				return
			}
			sendErr := grpcStream.Send(&api.ServerToClient{
				Message: &api.ServerToClient_Data{
					Data: &api.Data{ConnectionId: connID, Chunk: buf[:n]},
				},
			})
			if sendErr != nil {
				return
			}
		}
	}()

	wg.Wait()
	log.Printf("Connection %s: proxy finished.", connID)
}
