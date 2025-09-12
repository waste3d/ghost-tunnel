package server

import (
	"log"
	"net"

	"github.com/waste3d/ghost-tunnel/api"
	tunnelgrpc "github.com/waste3d/ghost-tunnel/internal/interfaces/grpc"
	"google.golang.org/grpc"
)

func main() {
	port := ":50051"
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	tunnelSrv := tunnelgrpc.NewTunnelService()

	api.RegisterTunnelServiceServer(grpcServer, tunnelSrv)

	log.Printf("server listening at %s", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
