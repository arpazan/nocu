// Package httpx holds the Echo server factory, middleware and response helpers
// shared by every HTTP service.
package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// Error is an HTTP-aware error. Handlers return it and ErrorHandler renders it.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	cause   error
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Unwrap exposes the wrapped cause for errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.cause }

// WithCause attaches an underlying error for logging without exposing it to clients.
func (e *Error) WithCause(err error) *Error {
	c := *e
	c.cause = err
	return &c
}

func newError(status int, code, msg string) *Error {
	return &Error{Status: status, Code: code, Message: msg}
}

// Constructors for the common cases.
func BadRequest(msg string) *Error   { return newError(http.StatusBadRequest, "bad_request", msg) }
func Unauthorized(msg string) *Error { return newError(http.StatusUnauthorized, "unauthorized", msg) }
func Forbidden(msg string) *Error    { return newError(http.StatusForbidden, "forbidden", msg) }
func NotFound(msg string) *Error     { return newError(http.StatusNotFound, "not_found", msg) }
func Conflict(msg string) *Error     { return newError(http.StatusConflict, "conflict", msg) }
func Internal(msg string) *Error {
	return newError(http.StatusInternalServerError, "internal_error", msg)
}

type errorBody struct {
	Error *Error `json:"error"`
}

// ErrorHandler renders every error as {"error":{"code","message"}} and logs 5xx.
func ErrorHandler(log *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		var out *Error
		var httpErr *echo.HTTPError
		switch {
		case errors.As(err, &out):
			// already shaped
		case errors.As(err, &httpErr):
			msg, _ := httpErr.Message.(string)
			if msg == "" {
				msg = http.StatusText(httpErr.Code)
			}
			out = newError(httpErr.Code, codeFor(httpErr.Code), msg)
		default:
			out = Internal("something went wrong").WithCause(err)
		}

		if out.Status >= 500 {
			log.ErrorContext(c.Request().Context(), "request failed",
				"status", out.Status, "code", out.Code, "err", err,
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
		}

		if c.Request().Method == http.MethodHead {
			_ = c.NoContent(out.Status)
			return
		}
		_ = c.JSON(out.Status, errorBody{Error: out})
	}
}

func codeFor(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusConflict:
		return "conflict"
	case http.StatusUnprocessableEntity:
		return "unprocessable"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		if status >= 500 {
			return "internal_error"
		}
		return "error"
	}
}
