package tunnelgrpc

import (
	"io"
	"log"

	"github.com/waste3d/ghost-tunnel/api"
)

type TunnelService struct {
	api.UnimplementedTunnelServiceServer
}

func NewTunnelService() *TunnelService {
	return &TunnelService{}
}

func (s *TunnelService) EstablishTunnel(stream api.TunnelService_EstablishTunnelServer) error {
	log.Println("Client connected")

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client disconnected gracefully")
			return nil
		}

		if err != nil {
			log.Println("Error receiving message:", err)
			return err
		}

		if reg := msg.GetRegister(); reg != nil {
			log.Printf("Client registered for tunnel ID: %s", reg.GetTunnelId())

			// В дальнейшем: проверить ауф, найти туннель по id, сохранить стрим в редис или мапу
		}

		// в дальнейшем: Обработать входящие данные (msg.GetData())
	}
}
