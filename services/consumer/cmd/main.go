// Command consumer persists broker events into the events table.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Pzquzer/Nocu/services/consumer/internal/config"
	"github.com/Pzquzer/Nocu/services/consumer/internal/consumer"
	sharedcfg "github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/database"
	"github.com/Pzquzer/Nocu/shared/logger"
	"github.com/Pzquzer/Nocu/shared/messaging"
)

func main() {
	if err := run(); err != nil {
		slog.Error("consumer exited with error", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := sharedcfg.Load[config.Config]()
	if err != nil {
		return err
	}
	log := logger.New(cfg.LogLevel, "consumer", cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, database.Options{DSN: cfg.DatabaseURL, Logger: log, LogQueries: cfg.IsDevelopment()})
	if err != nil {
		return err
	}
	defer database.Close(db)

	broker, err := messaging.ConnectNATS(ctx, messaging.NATSOptions{URL: cfg.NATSURL, Name: "consumer", Logger: log})
	if err != nil {
		return err
	}
	defer broker.Close()

	handler := consumer.NewHandler(consumer.NewGormRepository(db), log)

	// One durable consumer per subject so each filter has its own cursor.
	for _, subject := range cfg.Subjects {
		subject = strings.TrimSpace(subject)
		if subject == "" {
			continue
		}
		durable := cfg.DurableName + "-" + durableSuffix(subject)
		sub, err := broker.Subscribe(ctx, subject, durable, handler.Handle)
		if err != nil {
			return err
		}
		defer sub.Unsubscribe()
		log.Info("subscribed", "subject", subject, "durable", durable)
	}

	<-ctx.Done()
	log.Info("consumer shutting down")
	return nil
}

// durableSuffix turns "incident.>" into "incident" so names stay valid for JetStream.
func durableSuffix(subject string) string {
	r := strings.NewReplacer(".", "-", ">", "all", "*", "any")
	return strings.Trim(r.Replace(subject), "-")
}
