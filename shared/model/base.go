// Package model holds the GORM structs shared by every service.
//
// Models are the single source of truth for the schema. Versioned migrations
// in shared/migration create and alter the tables.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base carries the columns every table has.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a UUID when the caller did not set one.
func (b *Base) BeforeCreate(_ *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// All returns every model, in dependency order, for AutoMigrate.
func All() []any {
	return []any{
		&Role{},
		&User{},
		&Incident{},
		&Event{},
		&ScheduledJob{},
	}
}
