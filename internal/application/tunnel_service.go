package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/waste3d/ghost-tunnel/internal/domain"
)

// DTO - Data Transfer Objects для передачи данных из/в слой интерфейсов
type CreateTunnelRequest struct {
	UserID    domain.UserID
	Subdomain string
	LocalPort int
}

type TunnelService struct {
	tunnelRepo domain.TunnelRepository // Зависимость от ИНТЕРФЕЙСА
}

// *** ИСПРАВЛЕННАЯ СИГНАТУРА КОНСТРУКТОРА ***
// Теперь он принимает ИНТЕРФЕЙС, а не конкретную структуру.
func NewTunnelService(tunnelRepo domain.TunnelRepository) *TunnelService {
	return &TunnelService{tunnelRepo: tunnelRepo}
}

func (s *TunnelService) CreateTunnel(ctx context.Context, req CreateTunnelRequest) (*domain.Tunnel, error) {
	if req.Subdomain == "" {
		req.Subdomain = fmt.Sprintf("random-%d", time.Now().UnixNano()%10000)
	}

	newTunnel := &domain.Tunnel{
		ID:     domain.TunnelID(uuid.New().String()),
		UserID: req.UserID,
		Endpoints: domain.Endpoint{
			Subdomain: req.Subdomain,
			Domain:    "waste3d.ru",
			Port:      80,
		},
		LocalTarget: domain.LocalTarget{
			Host: "localhost",
			Port: req.LocalPort,
		},
		Status:    domain.StatusInactive,
		CreatedAt: time.Now(),
	}

	if err := s.tunnelRepo.Save(ctx, newTunnel); err != nil {
		return nil, fmt.Errorf("failed to create tunnel: %w", err)
	}

	return newTunnel, nil
}

func (s *TunnelService) DeleteTunnel(ctx context.Context, subdomain string) error {
	return s.tunnelRepo.Delete(ctx, subdomain)
}
