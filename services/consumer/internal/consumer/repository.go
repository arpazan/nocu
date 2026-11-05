// Package consumer persists broker messages as Event rows.
package consumer

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// EventRepository is the persistence boundary for events.
type EventRepository interface {
	Create(ctx context.Context, ev *model.Event) error
}

type gormRepository struct{ db *gorm.DB }

// NewGormRepository returns an EventRepository backed by GORM.
func NewGormRepository(db *gorm.DB) EventRepository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, ev *model.Event) error {
	if err := r.db.WithContext(ctx).Create(ev).Error; err != nil {
		return fmt.Errorf("consumer: create event: %w", err)
	}
	return nil
}
