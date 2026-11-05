// Command migrate applies, rolls back or lists the schema migrations.
//
//	migrate up      apply every pending migration
//	migrate down    roll back the most recent migration
//	migrate status  list migrations and whether each is applied
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/database"
	"github.com/Pzquzer/Nocu/shared/logger"
	"github.com/Pzquzer/Nocu/shared/migration"
)

type cfg struct {
	config.Base
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: migrate <up|down|status>\n")
	}
	flag.Parse()
	cmd := flag.Arg(0)
	if cmd == "" {
		flag.Usage()
		os.Exit(2)
	}
	if err := run(cmd); err != nil {
		slog.Error("migrate failed", "command", cmd, "err", err)
		os.Exit(1)
	}
}

func run(cmd string) error {
	c, err := config.Load[cfg]()
	if err != nil {
		return err
	}
	log := logger.New(c.LogLevel, "migrate", c.Env)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.Open(ctx, database.Options{DSN: c.DatabaseURL, Logger: log, MaxOpenConns: 2})
	if err != nil {
		return err
	}
	defer database.Close(db)

	switch cmd {
	case "up":
		if err := migration.Up(db); err != nil {
			return err
		}
		log.Info("migrations applied")
	case "down":
		if err := migration.Down(db); err != nil {
			return err
		}
		log.Info("last migration rolled back")
	case "status":
		entries, err := migration.Status(db)
		if err != nil {
			return err
		}
		for _, e := range entries {
			state := "pending"
			if e.Applied {
				state = "applied"
			}
			fmt.Printf("%-40s %s\n", e.ID, state)
		}
	default:
		return fmt.Errorf("unknown command %q (want up, down or status)", cmd)
	}
	return nil
}
