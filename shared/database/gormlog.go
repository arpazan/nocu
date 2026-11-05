package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

// slogLogger bridges GORM's logger interface onto slog.
type slogLogger struct {
	log        *slog.Logger
	level      gormlogger.LogLevel
	slow       time.Duration
	logQueries bool
}

func newSlogLogger(log *slog.Logger, slow time.Duration, logQueries bool) gormlogger.Interface {
	return &slogLogger{log: log.With("component", "gorm"), level: gormlogger.Warn, slow: slow, logQueries: logQueries}
}

func (l *slogLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	c := *l
	c.level = level
	return &c
}

func (l *slogLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Info {
		l.log.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Warn {
		l.log.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level >= gormlogger.Error {
		l.log.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= gormlogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	sql, rows := fc()
	attrs := []any{"elapsed", elapsed, "rows", rows, "sql", sql}

	switch {
	case err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound):
		l.log.ErrorContext(ctx, "query failed", append(attrs, "err", err)...)
	case l.slow > 0 && elapsed > l.slow:
		l.log.WarnContext(ctx, "slow query", attrs...)
	case l.logQueries:
		l.log.DebugContext(ctx, "query", attrs...)
	}
}
