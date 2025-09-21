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

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Save(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, api_key, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.APIKey,
		user.CreatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: %s", domain.ErrUserAlreadyExists, pgErr.Message)
		}
		return fmt.Errorf("could not save user: %w", err)
	}
	return nil
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `select id, email, password_hash, api_key, created_at from users where email = $1`
	row := r.db.QueryRow(ctx, query, email)

	return r.ScanUser(row)
}

func (r *PostgresUserRepository) FindByAPIKey(ctx context.Context, apiKey string) (*domain.User, error) {
	query := `select id, email, password_hash, api_key, created_at from users where api_key = $1`
	row := r.db.QueryRow(ctx, query, apiKey)

	return r.ScanUser(row)
}

func (r *PostgresUserRepository) ScanUser(row pgx.Row) (*domain.User, error) {
	var user domain.User

	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.APIKey,
		&user.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("could not scan user: %w", err)
	}

	return &user, nil
}
