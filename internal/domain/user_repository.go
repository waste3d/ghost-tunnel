package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByAPIKey(ctx context.Context, apiKey string) (*User, error)
}
