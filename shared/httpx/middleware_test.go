package httpx

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/auth"
	"github.com/Pzquzer/Nocu/shared/config"
)

func newTestEcho() (*echo.Echo, *auth.Manager) {
	tokens := auth.NewManager(config.JWT{Secret: "secret-secret-secret", Issuer: "t", AccessTTL: time.Minute, RefreshTTL: time.Hour})
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(slog.Default())
	g := e.Group("/p", JWTAuth(tokens))
	g.GET("/me", func(c echo.Context) error {
		claims, _ := ClaimsFrom(c)
		return OK(c, claims.Email)
	})
	g.GET("/admin", func(c echo.Context) error { return NoContent(c) }, RequireRole("admin"))
	return e, tokens
}

func do(e *echo.Echo, target, header string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if header != "" {
		req.Header.Set(echo.HeaderAuthorization, header)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	e, _ := newTestEcho()
	if rec := do(e, "/p/me", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestJWTAuthAcceptsHeaderAndQuery(t *testing.T) {
	e, tokens := newTestEcho()
	pair, _ := tokens.Issue(uuid.New(), "x@y.z", []string{"viewer"})

	if rec := do(e, "/p/me", "Bearer "+pair.AccessToken); rec.Code != http.StatusOK {
		t.Fatalf("header auth: got %d body %s", rec.Code, rec.Body.String())
	}
	if rec := do(e, "/p/me?token="+pair.AccessToken, ""); rec.Code != http.StatusOK {
		t.Fatalf("query auth: got %d body %s", rec.Code, rec.Body.String())
	}
}

func TestRequireRole(t *testing.T) {
	e, tokens := newTestEcho()
	viewer, _ := tokens.Issue(uuid.New(), "v@y.z", []string{"viewer"})
	admin, _ := tokens.Issue(uuid.New(), "a@y.z", []string{"admin"})

	if rec := do(e, "/p/admin", "Bearer "+viewer.AccessToken); rec.Code != http.StatusForbidden {
		t.Fatalf("viewer: got %d", rec.Code)
	}
	if rec := do(e, "/p/admin", "Bearer "+admin.AccessToken); rec.Code != http.StatusNoContent {
		t.Fatalf("admin: got %d", rec.Code)
	}
}

func TestErrorHandlerShapesUnknownErrors(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(slog.Default())
	e.GET("/boom", func(c echo.Context) error { return echo.ErrNotFound })
	rec := do(e, "/boom", "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d", rec.Code)
	}
	if body := rec.Body.String(); body != "{\"error\":{\"code\":\"not_found\",\"message\":\"Not Found\"}}\n" {
		t.Fatalf("unexpected body: %s", body)
	}
}
