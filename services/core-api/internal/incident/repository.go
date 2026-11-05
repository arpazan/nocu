// Package incident implements the incident CRUD exposed by core-api.
package incident

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Sentinel errors.
var (
	ErrNotFound          = errors.New("incident: not found")
	ErrValidation        = errors.New("incident: validation failed")
	ErrInvalidTransition = errors.New("incident: invalid status transition")
)

// ListFilter narrows List.
type ListFilter struct {
	Status   model.IncidentStatus
	Severity model.Severity
	Limit    int
	Offset   int
}

func (f *ListFilter) normalise() {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// Repository is the persistence boundary for incidents.
type Repository interface {
	Create(ctx context.Context, in *model.Incident) error
	Get(ctx context.Context, id uuid.UUID) (*model.Incident, error)
	List(ctx context.Context, f ListFilter) ([]model.Incident, int64, error)
	Update(ctx context.Context, in *model.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type gormRepository struct{ db *gorm.DB }

// NewGormRepository returns a Repository backed by GORM.
func NewGormRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, in *model.Incident) error {
	if err := r.db.WithContext(ctx).Create(in).Error; err != nil {
		return fmt.Errorf("incident: create: %w", err)
	}
	return nil
}

func (r *gormRepository) Get(ctx context.Context, id uuid.UUID) (*model.Incident, error) {
	var in model.Incident
	err := r.db.WithContext(ctx).First(&in, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("incident: get: %w", err)
	}
	return &in, nil
}

func (r *gormRepository) List(ctx context.Context, f ListFilter) ([]model.Incident, int64, error) {
	f.normalise()
	q := r.db.WithContext(ctx).Model(&model.Incident{})
	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}
	if f.Severity != "" {
		q = q.Where("severity = ?", f.Severity)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("incident: count: %w", err)
	}
	var items []model.Incident
	if err := q.Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("incident: list: %w", err)
	}
	return items, total, nil
}

func (r *gormRepository) Update(ctx context.Context, in *model.Incident) error {
	res := r.db.WithContext(ctx).Model(in).Select("*").Omit("id", "created_at").Updates(in)
	if res.Error != nil {
		return fmt.Errorf("incident: update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&model.Incident{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("incident: delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
