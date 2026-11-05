// Package logger configures the process-wide slog logger.
package logger

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

// New builds a logger for the given level and service and installs it as slog's default.
// Development uses a colorised, human-readable handler (tint); every other env emits JSON.
func New(level, service, env string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(strings.ToUpper(level))); err != nil {
		lvl = slog.LevelInfo
	}

	var handler slog.Handler
	if env == "development" {
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			Level:      lvl,
			TimeFormat: time.Kitchen,
			NoColor:    !isatty.IsTerminal(os.Stdout.Fd()),
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	}

	log := slog.New(handler).With("service", service, "env", env)
	slog.SetDefault(log)
	return log
}
