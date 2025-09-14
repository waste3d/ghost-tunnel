package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/waste3d/ghost-tunnel/internal/domain"
)

var ErrSubdomainTaken = errors.New("subdomain is already taken")

type PostgresTunnelRepository struct {
	db *pgxpool.Pool
}

func NewPostgresTunnelRepository(db *pgxpool.Pool) *PostgresTunnelRepository {
	return &PostgresTunnelRepository{db: db}
}

func (r *PostgresTunnelRepository) Save(ctx context.Context, tunnel *domain.Tunnel) error {
	query := `
		insert into tunnels (id, user_id, subdomain, local_host, local_port, status, created_at)
		values ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query, tunnel.ID, tunnel.UserID, tunnel.Endpoints[0].Subdomain, tunnel.LocalTarget.Host, tunnel.LocalTarget.Port, tunnel.Status, tunnel.CreatedAt)
	var pgErr *pgconn.PgError
	if err != nil {
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrSubdomainTaken
		}
		return fmt.Errorf("could not save tunnel: %w", err)
	}

	return nil
}

func (r *PostgresTunnelRepository) FindBySubdomain(ctx context.Context, subdomain string) (*domain.Tunnel, error) {
	query := `
		select id, user_id, subdomain, local_host, local_port, status, created_at
		from tunnels
		where subdomain = $1
	`
	row := r.db.QueryRow(ctx, query, subdomain)

	var t domain.Tunnel
	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.Endpoints[0].Subdomain,
		&t.LocalTarget.Host,
		&t.LocalTarget.Port,
		&t.Status,
		&t.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { // <-- Используем ошибку из pgx
			return nil, nil
		}
		return nil, fmt.Errorf("could not find tunnel by subdomain: %w", err)
	}
	t.Endpoints[0].Domain = "waste3d.ru"
	return &t, nil
}
