package identity

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/auth"
	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/model"
	"github.com/Pzquzer/Nocu/shared/testutil"
)

type handlerFixture struct {
	e      *echo.Echo
	repo   *fakeRepo // reached into only to deactivate a user, which has no endpoint
	tokens *auth.Manager
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	svc, repo, tokens := newService(t)

	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler(slog.Default())
	NewHandler(svc).Register(e.Group("/api"), httpx.JWTAuth(tokens))
	return &handlerFixture{e: e, repo: repo, tokens: tokens}
}

func (f *handlerFixture) do(method, path, token, body string) *httptest.ResponseRecorder {
	return testutil.Do(f.e, method, path, token, body)
}

const (
	registerBody = `{"email":" Alice@Example.com ","password":"correct horse","name":"Alice"}`
	loginBody    = `{"email":"alice@example.com","password":"correct horse"}`
)

func (f *handlerFixture) registerAlice(t *testing.T) UserResponse {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/auth/register", "", registerBody)
	return testutil.DecodeData[UserResponse](t, rec, http.StatusCreated)
}

func (f *handlerFixture) loginAlice(t *testing.T) auth.TokenPair {
	t.Helper()
	f.registerAlice(t)
	rec := f.do(http.MethodPost, "/api/auth/login", "", loginBody)
	return testutil.DecodeData[auth.TokenPair](t, rec, http.StatusOK)
}

func TestHandlerRegister(t *testing.T) {
	t.Run("success shapes the public user", func(t *testing.T) {
		f := newHandlerFixture(t)
		u := f.registerAlice(t)

		if u.Email != "alice@example.com" {
			t.Errorf("email = %q; want normalised", u.Email)
		}
		if u.Name != "Alice" || !u.IsActive || u.ID == uuid.Nil {
			t.Errorf("unexpected user: %+v", u)
		}
		if len(u.Roles) != 1 || u.Roles[0] != model.RoleViewer {
			t.Errorf("roles = %v; want [viewer]", u.Roles)
		}
		if u.CreatedAt.IsZero() {
			t.Error("created_at should be set")
		}
	})

	t.Run("response never carries password material", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodPost, "/api/auth/register", "", registerBody)
		if rec.Code != http.StatusCreated {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		if strings.Contains(strings.ToLower(rec.Body.String()), "password") {
			t.Errorf("response leaks password material: %s", rec.Body.String())
		}
	})

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{"malformed JSON", `{"email":`, http.StatusBadRequest, "invalid request body"},
		{"invalid email", `{"email":"nope","password":"correct horse"}`, http.StatusBadRequest, "invalid email"},
		{"short password", `{"email":"a@b.co","password":"short"}`, http.StatusBadRequest, "password must be at least 8 characters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			testutil.AssertError(t, f.do(http.MethodPost, "/api/auth/register", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}

	t.Run("duplicate email is a conflict", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.registerAlice(t)
		testutil.AssertError(t, f.do(http.MethodPost, "/api/auth/register", "", registerBody), http.StatusConflict, "email already registered")
	})
}

func TestHandlerLogin(t *testing.T) {
	t.Run("success returns a usable token pair", func(t *testing.T) {
		f := newHandlerFixture(t)
		u := f.registerAlice(t)
		pair := testutil.DecodeData[auth.TokenPair](t, f.do(http.MethodPost, "/api/auth/login", "", loginBody), http.StatusOK)

		claims, err := f.tokens.Parse(pair.AccessToken, auth.TokenAccess)
		if err != nil {
			t.Fatalf("access token invalid: %v", err)
		}
		if claims.UserID != u.ID || claims.Email != u.Email {
			t.Errorf("claims = %+v; want user %s", claims, u.ID)
		}
		if _, err := f.tokens.Parse(pair.RefreshToken, auth.TokenRefresh); err != nil {
			t.Errorf("refresh token invalid: %v", err)
		}
	})

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{"malformed JSON", `{`, http.StatusBadRequest, "invalid request body"},
		{"wrong password", `{"email":"alice@example.com","password":"wrong horse"}`, http.StatusUnauthorized, "invalid credentials"},
		{"unknown email", `{"email":"nobody@example.com","password":"correct horse"}`, http.StatusUnauthorized, "invalid credentials"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			f.registerAlice(t)
			testutil.AssertError(t, f.do(http.MethodPost, "/api/auth/login", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}

	t.Run("inactive account is forbidden", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.registerAlice(t)
		f.repo.users["alice@example.com"].IsActive = false
		testutil.AssertError(t, f.do(http.MethodPost, "/api/auth/login", "", loginBody), http.StatusForbidden, "account is inactive")
	})
}

func TestHandlerRefresh(t *testing.T) {
	t.Run("success issues a new pair", func(t *testing.T) {
		f := newHandlerFixture(t)
		pair := f.loginAlice(t)
		rec := f.do(http.MethodPost, "/api/auth/refresh", "", `{"refresh_token":"`+pair.RefreshToken+`"}`)
		next := testutil.DecodeData[auth.TokenPair](t, rec, http.StatusOK)
		if _, err := f.tokens.Parse(next.AccessToken, auth.TokenAccess); err != nil {
			t.Errorf("new access token invalid: %v", err)
		}
	})

	t.Run("access token is not accepted as refresh token", func(t *testing.T) {
		f := newHandlerFixture(t)
		pair := f.loginAlice(t)
		rec := f.do(http.MethodPost, "/api/auth/refresh", "", `{"refresh_token":"`+pair.AccessToken+`"}`)
		testutil.AssertError(t, rec, http.StatusUnauthorized, "invalid credentials")
	})

	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{"malformed JSON", `{`, http.StatusBadRequest, "invalid request body"},
		{"missing token", `{}`, http.StatusBadRequest, "refresh_token is required"},
		{"garbage token", `{"refresh_token":"not.a.jwt"}`, http.StatusUnauthorized, "invalid credentials"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			testutil.AssertError(t, f.do(http.MethodPost, "/api/auth/refresh", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}
}

func TestHandlerMe(t *testing.T) {
	t.Run("returns the authenticated user", func(t *testing.T) {
		f := newHandlerFixture(t)
		u := f.registerAlice(t)
		token := testutil.IssueToken(t, f.tokens, u.ID, u.Roles...)
		got := testutil.DecodeData[UserResponse](t, f.do(http.MethodGet, "/api/me", token, ""), http.StatusOK)
		if got.ID != u.ID || got.Email != u.Email {
			t.Errorf("got %+v; want %+v", got, u)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		f := newHandlerFixture(t)
		testutil.AssertError(t, f.do(http.MethodGet, "/api/me", "", ""), http.StatusUnauthorized, "missing credentials")
	})

	t.Run("valid token for deleted user", func(t *testing.T) {
		f := newHandlerFixture(t)
		token := testutil.IssueToken(t, f.tokens, uuid.New(), model.RoleViewer)
		testutil.AssertError(t, f.do(http.MethodGet, "/api/me", token, ""), http.StatusNotFound, "user not found")
	})
}

func TestToUserResponse(t *testing.T) {
	u := &model.User{Email: "x@y.z", Name: "X", IsActive: true}
	u.ID = uuid.New()
	u.Roles = []model.Role{{Name: model.RoleAdmin}, {Name: model.RoleViewer}}

	got := toUserResponse(u)
	if got.ID != u.ID || got.Email != u.Email || got.Name != u.Name || !got.IsActive {
		t.Errorf("fields not copied: %+v", got)
	}
	if len(got.Roles) != 2 || got.Roles[0] != model.RoleAdmin || got.Roles[1] != model.RoleViewer {
		t.Errorf("roles = %v", got.Roles)
	}

	empty := toUserResponse(&model.User{})
	if empty.Roles == nil || len(empty.Roles) != 0 {
		t.Errorf("roles for user without roles = %#v; want empty non-nil slice", empty.Roles)
	}
}
