// Package config loads service configuration from environment variables.
//
// Every service embeds Base and adds the blocks it needs (HTTP, JWT, ...).
// Values are read once at startup with Load; there is no runtime reload.
package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

// Base holds the settings every service in the monorepo needs.
type Base struct {
	Env             string        `env:"APP_ENV" envDefault:"development"`
	ServiceName     string        `env:"SERVICE_NAME" envDefault:"nocu"`
	LogLevel        string        `env:"LOG_LEVEL" envDefault:"info"`
	DatabaseURL     string        `env:"DATABASE_URL,required"`
	NATSURL         string        `env:"NATS_URL" envDefault:"nats://localhost:4222"`
	ShutdownTimeout time.Duration `env:"SHUTDOWN_TIMEOUT" envDefault:"10s"`
}

// IsProduction reports whether the service runs with APP_ENV=production.
func (b Base) IsProduction() bool { return b.Env == "production" }

// IsDevelopment reports whether the service runs with APP_ENV=development.
func (b Base) IsDevelopment() bool { return b.Env == "development" }

// HTTP holds settings for services that expose an HTTP listener.
type HTTP struct {
	Port         int           `env:"HTTP_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"HTTP_READ_TIMEOUT" envDefault:"15s"`
	WriteTimeout time.Duration `env:"HTTP_WRITE_TIMEOUT" envDefault:"30s"`
	CORSOrigins  []string      `env:"CORS_ORIGINS" envDefault:"*" envSeparator:","`
}

// JWT holds settings shared by the token issuer (identity) and verifiers (core-api).
type JWT struct {
	Secret     string        `env:"JWT_SECRET,required"`
	Issuer     string        `env:"JWT_ISSUER" envDefault:"nocu-identity"`
	AccessTTL  time.Duration `env:"JWT_ACCESS_TTL" envDefault:"15m"`
	RefreshTTL time.Duration `env:"JWT_REFRESH_TTL" envDefault:"168h"`
}

// Load parses environment variables into a config struct of type T.
func Load[T any]() (T, error) {
	var cfg T
	if err := env.Parse(&cfg); err != nil {
		return cfg, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// MustLoad is Load that panics on error. Use it only in main.
func MustLoad[T any]() T {
	cfg, err := Load[T]()
	if err != nil {
		panic(err)
	}
	return cfg
}
