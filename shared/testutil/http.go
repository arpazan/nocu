// Package testutil holds helpers shared by handler tests across services:
// a JWT manager with a fixed secret, request execution against an Echo
// instance, and decoding of the httpx response envelope.
//
// It is imported only from _test.go files.
package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/auth"
	"github.com/Pzquzer/Nocu/shared/config"
	"github.com/Pzquzer/Nocu/shared/httpx"
)

// NewTokenManager returns an auth.Manager with a fixed test secret and short TTLs.
func NewTokenManager() *auth.Manager {
	return auth.NewManager(config.JWT{
		Secret:     "unit-test-secret-value",
		Issuer:     "t",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
}

// IssueToken returns an access token for id holding exactly the given roles.
func IssueToken(t *testing.T, tokens *auth.Manager, id uuid.UUID, roles ...string) string {
	t.Helper()
	email := "user@example.com"
	if len(roles) > 0 {
		email = roles[0] + "@example.com"
	}
	pair, err := tokens.Issue(id, email, roles)
	if err != nil {
		t.Fatalf("issue token for %v: %v", roles, err)
	}
	return pair.AccessToken
}

// Do executes one JSON request against e. An empty token sends no
// Authorization header; an empty body sends an empty request body.
func Do(e *echo.Echo, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// Envelope mirrors the JSON shape written by httpx response helpers and ErrorHandler.
type Envelope struct {
	Data  json.RawMessage `json:"data"`
	Meta  *httpx.Meta     `json:"meta"`
	Error *httpx.Error    `json:"error"`
}

// Decode parses the response body as an Envelope.
func Decode(t *testing.T, rec *httptest.ResponseRecorder) Envelope {
	t.Helper()
	var env Envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return env
}

// DecodeData asserts the response status and parses the envelope's data as T.
func DecodeData[T any](t *testing.T, rec *httptest.ResponseRecorder, wantStatus int) T {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("got %d; want %d (body %s)", rec.Code, wantStatus, rec.Body.String())
	}
	var out T
	if err := json.Unmarshal(Decode(t, rec).Data, &out); err != nil {
		t.Fatalf("decode data as %T: %v", out, err)
	}
	return out
}

// DecodeList asserts a 200 and returns the raw data plus pagination meta.
func DecodeList(t *testing.T, rec *httptest.ResponseRecorder) (json.RawMessage, httpx.Meta) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d; want 200 (body %s)", rec.Code, rec.Body.String())
	}
	env := Decode(t, rec)
	if env.Meta == nil {
		t.Fatalf("response has no meta: %s", rec.Body.String())
	}
	return env.Data, *env.Meta
}

// AssertError checks the status and the error message in the envelope.
func AssertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("got %d; want %d (body %s)", rec.Code, wantStatus, rec.Body.String())
	}
	env := Decode(t, rec)
	if env.Error == nil || env.Error.Message != wantMessage {
		t.Errorf("error = %+v; want message %q", env.Error, wantMessage)
	}
}
