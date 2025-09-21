package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	userRepo   domain.UserRepository
}

// *** ИСПРАВЛЕННАЯ СИГНАТУРА КОНСТРУКТОРА ***
// Теперь он принимает ИНТЕРФЕЙС, а не конкретную структуру.
func NewTunnelService(tunnelRepo domain.TunnelRepository, userRepo domain.UserRepository) *TunnelService {
	return &TunnelService{tunnelRepo: tunnelRepo, userRepo: userRepo}
}

func (s *TunnelService) GetUserRepository() domain.UserRepository {
	return s.userRepo
}

func (s *TunnelService) CreateTunnel(ctx context.Context, req CreateTunnelRequest) (*domain.Tunnel, error) {
	if req.Subdomain == "" {
		randomBytes := make([]byte, 5)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, fmt.Errorf("failed to generate random subdomain: %w", err)
		}
		req.Subdomain = hex.EncodeToString(randomBytes)
	}

	newTunnel := &domain.Tunnel{
		ID:     domain.TunnelID(uuid.New().String()),
		UserID: "",
		Endpoints: domain.Endpoint{
			Subdomain: req.Subdomain,
			Domain:    "gtunnel.ru",
			Port:      80,
		},
		LocalTarget: domain.LocalTarget{
			Host: "localhost",
			Port: req.LocalPort,
		},
		Status:    domain.StatusInactive,
		CreatedAt: time.Now(),
	}

	newTunnel.UserID = req.UserID

	if err := s.tunnelRepo.Save(ctx, newTunnel); err != nil {
		return nil, fmt.Errorf("failed to create tunnel: %w", err)
	}

	return newTunnel, nil
}

func (s *TunnelService) DeleteTunnel(ctx context.Context, subdomain string) error {
	return s.tunnelRepo.Delete(ctx, subdomain)
}
