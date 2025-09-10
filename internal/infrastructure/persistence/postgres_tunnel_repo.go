package persistence

import (
	"context"
	"database/sql"

	"github.com/waste3d/ghost-tunnel/internal/domain"
)

type PostgresTunnelRepository struct {
	db *sql.DB
}

func NewPostgresTunnelRepository(db *sql.DB) *PostgresTunnelRepository {
	return &PostgresTunnelRepository{db: db}
}

func (r *PostgresTunnelRepository) Save(ctx context.Context, tunnel *domain.Tunnel) error {
	return nil
}

func (r *PostgresTunnelRepository) FindByID(ctx context.Context, id domain.TunnelID) (*domain.Tunnel, error) {
	return nil, nil
}

func (r *PostgresTunnelRepository) FindBySubdomain(ctx context.Context, subdomain string) (*domain.Tunnel, error) {
	return nil, nil
}

func (r *PostgresTunnelRepository) Delete(ctx context.Context, id domain.TunnelID) error {
	return nil
}
