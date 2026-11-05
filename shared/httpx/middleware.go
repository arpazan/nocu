package httpx

import (
	"log/slog"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/Pzquzer/Nocu/shared/auth"
)

const claimsKey = "nocu.claims"

// RequestLogger logs one line per request with slog.
func RequestLogger(log *slog.Logger) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:   true,
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogRequestID: true,
		LogRemoteIP:  true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method,
				"uri", v.URI,
				"status", v.Status,
				"latency", v.Latency.Round(time.Microsecond),
				"request_id", v.RequestID,
				"remote_ip", v.RemoteIP,
			}
			if v.Error != nil {
				attrs = append(attrs, "err", v.Error)
			}
			switch {
			case v.Status >= 500:
				log.ErrorContext(c.Request().Context(), "request", attrs...)
			case v.Status >= 400:
				log.WarnContext(c.Request().Context(), "request", attrs...)
			default:
				log.InfoContext(c.Request().Context(), "request", attrs...)
			}
			return nil
		},
	})
}

// JWTAuth verifies an access token from the Authorization header or, for
// WebSocket clients that cannot set headers, the "token" query parameter.
func JWTAuth(tokens *auth.Manager) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			raw := bearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
			if raw == "" {
				raw = c.QueryParam("token")
			}
			if raw == "" {
				return Unauthorized("missing credentials")
			}
			claims, err := tokens.Parse(raw, auth.TokenAccess)
			if err != nil {
				return Unauthorized("invalid or expired token").WithCause(err)
			}
			c.Set(claimsKey, claims)
			return next(c)
		}
	}
}

// RequireRole allows the request through only when the subject holds at least one role.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := ClaimsFrom(c)
			if !ok {
				return Unauthorized("missing credentials")
			}
			for _, r := range roles {
				if claims.HasRole(r) {
					return next(c)
				}
			}
			return Forbidden("insufficient role")
		}
	}
}

// ClaimsFrom returns the claims stored by JWTAuth.
func ClaimsFrom(c echo.Context) (*auth.Claims, bool) {
	claims, ok := c.Get(claimsKey).(*auth.Claims)
	return claims, ok
}

func bearerToken(header string) string {
	const prefix = "bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}
