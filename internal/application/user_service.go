package application

import (
	"context"
	"fmt"

	"github.com/waste3d/ghost-tunnel/internal/domain"
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
		return nil, domain.ErrUserAlreadyExists
	}

	if req.Password != req.ConfirmPassword {
		return nil, domain.ErrPasswordMismatch
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

// DTO для авторизации
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *UserService) Login(ctx context.Context, req LoginRequest) (*domain.User, error) {
	user, err := s.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	if user == nil {
		return nil, domain.ErrInvalidCredentials
	}

	if !user.VerifyPassword(req.Password) {
		return nil, domain.ErrInvalidCredentials
	}

	return user, nil
}
