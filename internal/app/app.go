package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
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

type App struct {
	grpcServer   *grpc.Server
	apiServer    *http.Server
	publicServer net.Listener
	dbpool       *pgxpool.Pool
}

func New(ctx context.Context) (*App, error) {
	// Инициализация базы данных
	connStr := "postgres://postgres:postgres@localhost:5432/ghost_tunnel?sslmode=disable"
	dbPool, err := initDB(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}

	// Инициализация компонентов
	sessionManager := tunnelgrpc.NewSessionManager()
	connManager := tunnelgrpc.NewConnectionManager()
	tunnelRepo := persistence.NewPostgresTunnelRepository(dbPool)
	tunnelService := application.NewTunnelService(tunnelRepo)
	tunnelHandler := http_handlers.NewTunnelHandler(tunnelService)

	// Инициализация серверов
	grpcServer := initGrpcServer(sessionManager, connManager)
	apiServer := initApiServer(tunnelHandler)
	publicServer, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatalf("Failed to listen on port 8000: %v", err)
	}

	go acceptPublicConnections(publicServer, sessionManager, connManager, tunnelRepo)

	return &App{
		grpcServer:   grpcServer,
		apiServer:    apiServer,
		publicServer: publicServer,
		dbpool:       dbPool,
	}, nil
}

func (a *App) Run() {
	wg := sync.WaitGroup{}
	wg.Add(2)

	// Запускаем gprc сервер
	go func() {
		log.Println("gRPC server listening on port 50051")
		lis, _ := net.Listen("tcp", ":50051")
		if err := a.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	// Запускаем API сервер
	go func() {
		log.Println("API server listening on port 8081")
		if err := a.apiServer.ListenAndServe(); err != nil {
			log.Fatalf("Failed to serve API server: %v", err)
		}
		wg.Done()
	}()

	log.Println("Public server listening on port 8000")

	// Реазилуем graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		a.grpcServer.GracefulStop()
		wg.Done()
	}()

	if err := a.apiServer.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown API server: %v", err)
	}

	if err := a.publicServer.Close(); err != nil {
		log.Fatalf("Failed to close public server: %v", err)
	}

	a.dbpool.Close()

	wg.Wait()
	log.Println("Server shutdown complete")
}

// Функции инициализаторы

func initDB(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %v", err)
	}

	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name: "uuid", OID: pgtype.UUIDOID, Codec: &pgtype.UUIDCodec{},
		})
		return nil
	}

	dbPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %v", err)
	}

	if err = dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}
	log.Println("Successfully connected to database using pgxpool.")

	return dbPool, nil
}

func initGrpcServer(sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager) *grpc.Server {
	grpcServer := grpc.NewServer()
	tunnelSrv := tunnelgrpc.NewTunnelServer(sm, connMgr)
	api.RegisterTunnelServiceServer(grpcServer, tunnelSrv)
	return grpcServer
}

func initApiServer(tunnelHandler *http_handlers.TunnelHandler) *http.Server {
	router := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4321", "https://gtunnel.ru"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}

	router.Use(cors.New(config))

	tunnelHandler.RegisterRoutes(router)

	return &http.Server{
		Addr:    ":8081",
		Handler: router,
	}
}

func acceptPublicConnections(lis net.Listener, sm *tunnelgrpc.SessionManager, connMgr *tunnelgrpc.ConnectionManager, tunnelRepo domain.TunnelRepository) {
	for {
		conn, err := lis.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				log.Println("Public server closed")
				return
			}
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
		return
	}

	subdomain := strings.Split(req.Host, ".")[0]
	tunnel, err := tunnelRepo.FindBySubdomain(context.Background(), subdomain)
	if err != nil || tunnel == nil {
		return
	}

	grpcStream, ok := sm.Get(string(tunnel.ID))
	if !ok {
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

	log.Printf("Connection %s: starting proxy for '%s'", connID, subdomain)

	var wg sync.WaitGroup
	wg.Add(2)

	// Горутина 1: gRPC -> publicConn (ответ браузеру)
	go func() {
		defer wg.Done()
		defer publicConn.Close()
		for data := range dataChan {
			if _, err := publicConn.Write(data); err != nil {
				return
			}
		}
	}()

	// Горутина 2: publicConn -> gRPC (запрос от браузера)
	go func() {
		defer wg.Done()

		req.Header.Set("Connection", "close")
		req.Close = true
		rawRequest, _ := httputil.DumpRequest(req, true)
		err := grpcStream.Send(&api.ServerToClient{
			Message: &api.ServerToClient_Data{
				Data: &api.Data{ConnectionId: connID, Chunk: rawRequest},
			},
		})
		if err != nil {
			close(dataChan)
			return
		}

		// Этот цикл теперь с таймаутом
		for {
			// Устанавливаем дедлайн на чтение. Если за 2 секунды ничего не придет, Read вернет ошибку.
			publicConn.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 4*1024)
			n, err := publicConn.Read(buf)
			if err != nil {
				// Таймаут или другая ошибка - сигнал к завершению.
				close(dataChan)
				return
			}
			// Если данные пришли, сбрасываем таймаут и отправляем их.
			publicConn.SetReadDeadline(time.Time{})
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

type StreamWriter struct {
	stream api.TunnelService_EstablishTunnelServer
	connID string
}

func (w *StreamWriter) Write(p []byte) (n int, err error) {
	err = w.stream.Send(&api.ServerToClient{
		Message: &api.ServerToClient_Data{
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
