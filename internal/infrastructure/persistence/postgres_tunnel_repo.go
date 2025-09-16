package persistence

import (
	"context"
	"database/sql"
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

// NewPostgresTunnelRepository теперь возвращает ИНТЕРФЕЙС, а не структуру.
// Это хорошая практика.
func NewPostgresTunnelRepository(db *pgxpool.Pool) domain.TunnelRepository {
	return &PostgresTunnelRepository{db: db}
}

func (r *PostgresTunnelRepository) Save(ctx context.Context, tunnel *domain.Tunnel) error {
	query := `
		INSERT INTO tunnels (id, user_id, subdomain, local_host, local_port, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	var userID interface{}
	if tunnel.UserID != "" {
		userID = tunnel.UserID
	} else {
		userID = nil
	}

	_, err := r.db.Exec(ctx, query,
		tunnel.ID,
		userID,
		tunnel.Endpoints.Subdomain,
		tunnel.LocalTarget.Host,
		tunnel.LocalTarget.Port,
		tunnel.Status,
		tunnel.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrSubdomainTaken
		}
		return fmt.Errorf("could not save tunnel: %w", err)
	}
	return nil
}

func (r *PostgresTunnelRepository) FindBySubdomain(ctx context.Context, subdomain string) (*domain.Tunnel, error) {
	query := `SELECT id, user_id, subdomain, local_host, local_port, status, created_at FROM tunnels WHERE subdomain = $1`
	row := r.db.QueryRow(ctx, query, subdomain)

	var t domain.Tunnel
	var userID sql.NullString

	err := row.Scan(
		&t.ID,
		&userID,
		&t.Endpoints.Subdomain,
		&t.LocalTarget.Host,
		&t.LocalTarget.Port,
		&t.Status,
		&t.CreatedAt,
	)

	if userID.Valid {
		t.UserID = domain.UserID(userID.String)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("could not find tunnel by subdomain: %w", err)
	}
	t.Endpoints.Domain = "waste3d.ru"
	return &t, nil
}

func (r *PostgresTunnelRepository) FindByID(ctx context.Context, id domain.TunnelID) (*domain.Tunnel, error) {
	return nil, errors.New("not implemented")
}

func (r *PostgresTunnelRepository) Delete(ctx context.Context, subdomain string) error {
	query := `DELETE FROM tunnels WHERE subdomain = $1`
	_, err := r.db.Exec(ctx, query, subdomain)
	if err != nil {
		return fmt.Errorf("could not delete tunnel: %w", err)
	}
	return nil
}
