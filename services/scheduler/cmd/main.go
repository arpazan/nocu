// Command scheduler runs the cron jobs and the uptime probe worker for the platform.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

	"github.com/Pzquzer/Nocu/services/scheduler/internal/config"
	"github.com/Pzquzer/Nocu/services/scheduler/internal/scheduler"
	"github.com/Pzquzer/Nocu/services/scheduler/internal/uptime"
	sharedcfg "github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/database"
	"github.com/Pzquzer/Nocu/shared/logger"
	"github.com/Pzquzer/Nocu/shared/messaging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("scheduler exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := sharedcfg.Load[config.Config]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "scheduler", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, database.Options{DSN: cfg.DatabaseURL, Logger: log, LogQueries: cfg.IsDevelopment()})
	if err != nil {
		return err
	}
	defer database.Close(db)

	broker, err := messaging.ConnectNATS(ctx, messaging.NATSOptions{URL: cfg.NATSURL, Name: "scheduler", Logger: log})
	if err != nil {
		return err
	}
	defer broker.Close()

	// Uptime monitoring: a tick-driven worker for probes plus a daily expiry sweep.
	store := uptime.NewGormStore(db)
	dispatcher := uptime.NewDispatcher(log, cfg.MonitorAllowPrivate)
	worker := uptime.NewWorker(store, uptime.NewProber(cfg.MonitorAllowPrivate), broker, dispatcher, log, uptime.WorkerOptions{
		Tick:        cfg.MonitorTick,
		Concurrency: cfg.MonitorConcurrency,
	})
	expiry := uptime.NewExpiryChecker(store, broker, dispatcher, log, uptime.ExpiryOptions{WarnDays: cfg.ExpiryWarnDays})

	runner := scheduler.NewRunner(db, broker, log, cfg.JobTimeout)
	jobs := scheduler.DefaultJobs(scheduler.JobDeps{
		DB:                 db,
		Pub:                broker,
		Log:                log,
		SLAAckDeadline:     cfg.SLAAckDeadline,
		EventRetention:     cfg.EventRetention,
		HeartbeatRetention: cfg.HeartbeatRetention,
		ExpiryCheck:        expiry.Run,
	})
	for _, job := range jobs {
		if err := runner.Register(job); err != nil {
			return err
		}
	}

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return runner.Start(gctx) })
	g.Go(func() error { return worker.Run(gctx) })
	return g.Wait()
}
