package application

import (
	"context"
	"errors"
	"fmt"

	"github.com/waste3d/ghost-tunnel/internal/domain"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPasswordMismatch   = errors.New("password and confirm password do not match")
)

type UserService struct {
	userRepo domain.UserRepository
}

func NewUserService(userRepo domain.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// DTO для регистрации
type RegisterRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (s *UserService) Register(ctx context.Context, req RegisterRequest) (*domain.User, error) {
	existingUser, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserAlreadyExists
	}

	if req.Password != req.ConfirmPassword {
		return nil, ErrPasswordMismatch
	}

	newUser, err := domain.NewUser(req.Email, req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.userRepo.Save(ctx, newUser); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return newUser, nil

}
