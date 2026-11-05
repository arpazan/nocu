// Package database opens and manages the shared GORM connection.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Options configures Open.
type Options struct {
	DSN             string
	Logger          *slog.Logger
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	SlowThreshold   time.Duration
	// LogQueries enables debug logging of every SQL statement.
	LogQueries bool
}

func (o *Options) defaults() {
	if o.Logger == nil {
		o.Logger = slog.Default()
	}
	if o.MaxOpenConns == 0 {
		o.MaxOpenConns = 20
	}
	if o.MaxIdleConns == 0 {
		o.MaxIdleConns = 5
	}
	if o.ConnMaxLifetime == 0 {
		o.ConnMaxLifetime = 30 * time.Minute
	}
	if o.SlowThreshold == 0 {
		o.SlowThreshold = 200 * time.Millisecond
	}
}

// Open connects to PostgreSQL, applies pool settings and verifies the connection.
func Open(ctx context.Context, opts Options) (*gorm.DB, error) {
	opts.defaults()

	db, err := gorm.Open(postgres.Open(opts.DSN), &gorm.Config{
		Logger:         newSlogLogger(opts.Logger, opts.SlowThreshold, opts.LogQueries),
		NowFunc:        func() time.Time { return time.Now().UTC() },
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return db, nil
}

// Ping checks the connection; use it for readiness probes.
func Ping(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close releases the underlying connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
