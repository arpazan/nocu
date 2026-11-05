package identity

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Pzquzer/Nocu/shared/auth"
	"github.com/Pzquzer/Nocu/shared/model"
)

// Service holds the identity business rules.
type Service struct {
	users             UserRepository
	tokens            *auth.Manager
	log               *slog.Logger
	defaultRole       string
	minPasswordLength int
	bcryptCost        int
}

// Options configures NewService.
type Options struct {
	DefaultRole       string
	MinPasswordLength int
	BcryptCost        int
}

// NewService wires the service.
func NewService(users UserRepository, tokens *auth.Manager, log *slog.Logger, opts Options) *Service {
	if opts.MinPasswordLength <= 0 {
		opts.MinPasswordLength = 8
	}
	if opts.BcryptCost <= 0 {
		opts.BcryptCost = bcrypt.DefaultCost
	}
	return &Service{
		users:             users,
		tokens:            tokens,
		log:               log,
		defaultRole:       opts.DefaultRole,
		minPasswordLength: opts.MinPasswordLength,
		bcryptCost:        opts.BcryptCost,
	}
}

// RegisterInput is what Register needs.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
}

// Register creates an account and grants it the default role.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if _, err := mail.ParseAddress(email); err != nil || email == "" {
		return nil, fmt.Errorf("%w: invalid email", ErrValidation)
	}
	if len(in.Password) < s.minPasswordLength {
		return nil, fmt.Errorf("%w: password must be at least %d characters", ErrValidation, s.minPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("identity: hash password: %w", err)
	}

	user := &model.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(in.Name),
		IsActive:     true,
	}
	if s.defaultRole != "" {
		role, err := s.users.FindRoleByName(ctx, s.defaultRole)
		switch {
		case err == nil:
			user.Roles = []model.Role{*role}
		case errors.Is(err, ErrNotFound):
			s.log.Warn("default role missing, registering without roles", "role", s.defaultRole)
		default:
			return nil, err
		}
	}

	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	s.log.Info("user registered", "user_id", user.ID, "email", user.Email)
	return user, nil
}

// Login verifies credentials and issues tokens.
func (s *Service) Login(ctx context.Context, email, password string) (auth.TokenPair, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, ErrNotFound) {
		// Burn roughly the same time as a real compare to blunt user enumeration.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi5XIhwQ9W7sJ1aKX1Y1V8r1KZyJ9A6"), []byte(password))
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return auth.TokenPair{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	if !user.IsActive {
		return auth.TokenPair{}, ErrInactive
	}
	return s.tokens.Issue(user.ID, user.Email, user.RoleNames())
}

// Refresh exchanges a refresh token for a new pair, re-reading the user so
// role or status changes take effect.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (auth.TokenPair, error) {
	claims, err := s.tokens.Parse(refreshToken, auth.TokenRefresh)
	if err != nil {
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	user, err := s.users.FindByID(ctx, claims.UserID)
	if errors.Is(err, ErrNotFound) {
		return auth.TokenPair{}, ErrInvalidCredentials
	}
	if err != nil {
		return auth.TokenPair{}, err
	}
	if !user.IsActive {
		return auth.TokenPair{}, ErrInactive
	}
	return s.tokens.Issue(user.ID, user.Email, user.RoleNames())
}

// Me loads the current user's profile.
func (s *Service) Me(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.users.FindByID(ctx, id)
}
