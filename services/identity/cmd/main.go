// Command identity runs the authentication and user service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Pzquzer/Nocu/services/identity/internal/config"
	"github.com/Pzquzer/Nocu/services/identity/internal/identity"
	"github.com/Pzquzer/Nocu/shared/auth"
	sharedcfg "github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/database"
	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/logger"
)

func main() {
	if err := run(); err != nil {
		slog.Error("identity exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := sharedcfg.Load[config.Config]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "identity", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, database.Options{DSN: cfg.DatabaseURL, Logger: log, LogQueries: cfg.IsDevelopment()})
	if err != nil {
		return err
	}
	defer database.Close(db)

	tokens := auth.NewManager(cfg.JWT)
	svc := identity.NewService(identity.NewGormRepository(db), tokens, log, identity.Options{
		DefaultRole:       cfg.DefaultRole,
		MinPasswordLength: cfg.MinPasswordLength,
	})

	e := httpx.NewServer(cfg.HTTP, log, map[string]httpx.ReadinessCheck{
		"database": func(ctx context.Context) error { return database.Ping(ctx, db) },
	})
	identity.NewHandler(svc).Register(e.Group("/api/v1"), httpx.JWTAuth(tokens))

	return httpx.Serve(ctx, e, cfg.Port, cfg.ShutdownTimeout, log)
}
