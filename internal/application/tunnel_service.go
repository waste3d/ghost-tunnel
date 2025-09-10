package application

import (
	"context"

	"github.com/google/uuid"
	"github.com/waste3d/ghost-tunnel/internal/domain"
)

// DTO - Data Transfer Objects для передачи данных из/в слой интерфейсов
type CreateTunnelRequest struct {
	UserID    domain.UserID
	Subdomain string // Если пустой - автоматически будем генерировать
	LocalPort int
}

type TunnelService struct {
	tunnelRepo domain.TunnelRepository
}

func NewTunnelService(tunnelRepo domain.TunnelRepository) *TunnelService {
	return &TunnelService{tunnelRepo: tunnelRepo}
}

func (s *TunnelService) CreateTunnel(ctx context.Context, req CreateTunnelRequest) (*domain.Tunnel, error) {
	newTunnel := &domain.Tunnel{
		ID:     domain.TunnelID(uuid.New().String()), // Генерируем ID для нового туннеля
		UserID: req.UserID,
		Endpoints: []domain.Endpoint{
			{
				Subdomain: req.Subdomain,
				Domain:    "domain.com",
				Port:      req.LocalPort,
			},
		},
		LocalTarget: domain.LocalTarget{
			Host: "0.0.0.0",
			Port: req.LocalPort,
		},
	}

	newTunnel.Deactivate()

	if err := s.tunnelRepo.Save(ctx, newTunnel); err != nil {
		return nil, err
	}

	return newTunnel, nil
}
