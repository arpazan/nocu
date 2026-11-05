// Command core-api runs the incident and monitor APIs and the WebSocket gateway.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/Pzquzer/Nocu/services/core-api/internal/config"
	"github.com/Pzquzer/Nocu/services/core-api/internal/incident"
	"github.com/Pzquzer/Nocu/services/core-api/internal/monitor"
	"github.com/Pzquzer/Nocu/services/core-api/internal/ws"
	"github.com/Pzquzer/Nocu/shared/auth"
	sharedcfg "github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/database"
	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/logger"
	"github.com/Pzquzer/Nocu/shared/messaging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("core-api exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := sharedcfg.Load[config.Config]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "core-api", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, database.Options{DSN: cfg.DatabaseURL, Logger: log, LogQueries: cfg.IsDevelopment()})
	if err != nil {
		return err
	}
	defer database.Close(db)

	broker, err := messaging.ConnectNATS(ctx, messaging.NATSOptions{URL: cfg.NATSURL, Name: "core-api", Logger: log})
	if err != nil {
		return err
	}
	defer broker.Close()

	tokens := auth.NewManager(cfg.JWT)
	hub := ws.NewHub(log, cfg.WSSendBuffer)
	defer hub.CloseAll()

	e := httpx.NewServer(cfg.HTTP, log, map[string]httpx.ReadinessCheck{
		"database": func(ctx context.Context) error { return database.Ping(ctx, db) },
		"nats":     broker.Ping,
	})
	api := e.Group("/api/v1", httpx.JWTAuth(tokens))
	incident.NewHandler(incident.NewService(incident.NewGormRepository(db), broker, log)).Register(api)
	monitor.NewHandler(monitor.NewService(monitor.NewGormRepository(db), broker, log)).Register(api)
	ws.NewHandler(hub, ws.Options{
		WriteTimeout:   cfg.WSWriteTimeout,
		PingInterval:   cfg.WSPingInterval,
		OriginPatterns: cfg.CORSOrigins,
	}).Register(api)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return ws.Bridge(gctx, broker, hub, messaging.SubjectIncidentAll, messaging.SubjectJobAll, messaging.SubjectMonitorAll)
	})
	g.Go(func() error {
		return httpx.Serve(gctx, e, cfg.Port, cfg.ShutdownTimeout, log)
	})
	return g.Wait()
}
