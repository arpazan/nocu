// Package identity implements user registration, login and token refresh.
package identity

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Sentinel errors returned by the repository and service.
var (
	ErrNotFound           = errors.New("identity: not found")
	ErrEmailTaken         = errors.New("identity: email already registered")
	ErrInvalidCredentials = errors.New("identity: invalid credentials")
	ErrInactive           = errors.New("identity: account inactive")
	ErrValidation         = errors.New("identity: validation failed")
)

// UserRepository is the persistence boundary for users and roles.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	FindRoleByName(ctx context.Context, name string) (*model.Role, error)
}

type gormRepository struct {
	db *gorm.DB
}

// NewGormRepository returns a UserRepository backed by GORM.
func NewGormRepository(db *gorm.DB) UserRepository {
	return &gormRepository{db: db}
}

func (r *gormRepository) Create(ctx context.Context, user *model.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("identity: create user: %w", err)
	}
	return nil
}

func (r *gormRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").Where("email = ?", email).First(&user).Error
	return r.wrap(&user, err)
}

func (r *gormRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error
	return r.wrap(&user, err)
}

func (r *gormRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("identity: find role: %w", err)
	}
	return &role, nil
}

func (r *gormRepository) wrap(user *model.User, err error) (*model.User, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("identity: find user: %w", err)
	}
	return user, nil
}
