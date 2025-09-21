package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordMismatch   = errors.New("password and confirm password do not match")
)

type UserID string
type APIKey string

type User struct {
	ID           UserID
	Email        string
	PasswordHash string
	APIKey       APIKey
	CreatedAt    time.Time
}

func NewUser(email, password string) (*User, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	return &User{
		ID:           UserID(uuid.New().String()),
		Email:        email,
		PasswordHash: string(hashedPassword),
		APIKey:       APIKey(apiKey),
		CreatedAt:    time.Now(),
	}, nil
}

func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

func generateAPIKey() (string, error) {
	return (uuid.New().String()), nil
}
