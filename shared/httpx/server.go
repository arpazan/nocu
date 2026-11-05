package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Pzquzer/Nocu/shared/config"
)

// ReadinessCheck reports whether a dependency is usable.
type ReadinessCheck func(ctx context.Context) error

// NewServer builds an Echo instance with the shared middleware stack and
// /healthz and /readyz endpoints.
func NewServer(cfg config.HTTP, log *slog.Logger, checks map[string]ReadinessCheck) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = ErrorHandler(log)
	e.Server.ReadTimeout = cfg.ReadTimeout
	e.Server.WriteTimeout = cfg.WriteTimeout

	e.Use(middleware.RequestID())
	e.Use(RequestLogger(log))
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.CORSOrigins,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
	}))

	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/readyz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 3*time.Second)
		defer cancel()
		result := map[string]string{}
		status := http.StatusOK
		for name, check := range checks {
			if err := check(ctx); err != nil {
				result[name] = err.Error()
				status = http.StatusServiceUnavailable
				continue
			}
			result[name] = "ok"
		}
		return c.JSON(status, result)
	})
	return e
}

// Serve runs the server until ctx is cancelled, then shuts it down gracefully.
func Serve(ctx context.Context, e *echo.Echo, port int, shutdownTimeout time.Duration, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("http server listening", "port", port)
		if err := e.Start(fmt.Sprintf(":%d", port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("httpx: serve: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	log.Info("http server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("httpx: shutdown: %w", err)
	}
	return nil
}
