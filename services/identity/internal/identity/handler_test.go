package identity

import (
	"encoding/json"
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
)

type handlerFixture struct {
	e      *echo.Echo
	repo   *fakeRepo
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
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	return rec
}

type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *httpx.Error    `json:"error"`
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	return env
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) UserResponse {
	t.Helper()
	var u UserResponse
	if err := json.Unmarshal(decode(t, rec).Data, &u); err != nil {
		t.Fatalf("decode user: %v", err)
	}
	return u
}

func decodePair(t *testing.T, rec *httptest.ResponseRecorder) auth.TokenPair {
	t.Helper()
	var p auth.TokenPair
	if err := json.Unmarshal(decode(t, rec).Data, &p); err != nil {
		t.Fatalf("decode token pair: %v", err)
	}
	return p
}

const registerBody = `{"email":" Alice@Example.com ","password":"correct horse","name":"Alice"}`

func (f *handlerFixture) registerAlice(t *testing.T) UserResponse {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/auth/register", "", registerBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: got %d body %s", rec.Code, rec.Body.String())
	}
	return decodeUser(t, rec)
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("got %d; want %d (body %s)", rec.Code, wantStatus, rec.Body.String())
	}
	env := decode(t, rec)
	if env.Error == nil || env.Error.Message != wantMessage {
		t.Errorf("error = %+v; want message %q", env.Error, wantMessage)
	}
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

		// The raw body must never carry the password hash.
		rec := f.do(http.MethodPost, "/api/auth/register", "", `{"email":"b@example.com","password":"correct horse","name":"B"}`)
		if strings.Contains(rec.Body.String(), "password") {
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
			assertError(t, f.do(http.MethodPost, "/api/auth/register", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}

	t.Run("duplicate email is a conflict", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.registerAlice(t)
		assertError(t, f.do(http.MethodPost, "/api/auth/register", "", registerBody), http.StatusConflict, "email already registered")
	})
}

func TestHandlerLogin(t *testing.T) {
	t.Run("success returns a usable token pair", func(t *testing.T) {
		f := newHandlerFixture(t)
		u := f.registerAlice(t)

		rec := f.do(http.MethodPost, "/api/auth/login", "", `{"email":"alice@example.com","password":"correct horse"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		pair := decodePair(t, rec)
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
			assertError(t, f.do(http.MethodPost, "/api/auth/login", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}

	t.Run("inactive account is forbidden", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.registerAlice(t)
		f.repo.users["alice@example.com"].IsActive = false
		assertError(t, f.do(http.MethodPost, "/api/auth/login", "", `{"email":"alice@example.com","password":"correct horse"}`),
			http.StatusForbidden, "account is inactive")
	})
}

func TestHandlerRefresh(t *testing.T) {
	login := func(t *testing.T, f *handlerFixture) auth.TokenPair {
		t.Helper()
		f.registerAlice(t)
		rec := f.do(http.MethodPost, "/api/auth/login", "", `{"email":"alice@example.com","password":"correct horse"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("login: got %d body %s", rec.Code, rec.Body.String())
		}
		return decodePair(t, rec)
	}

	t.Run("success issues a new pair", func(t *testing.T) {
		f := newHandlerFixture(t)
		pair := login(t, f)
		rec := f.do(http.MethodPost, "/api/auth/refresh", "", `{"refresh_token":"`+pair.RefreshToken+`"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		next := decodePair(t, rec)
		if _, err := f.tokens.Parse(next.AccessToken, auth.TokenAccess); err != nil {
			t.Errorf("new access token invalid: %v", err)
		}
	})

	t.Run("access token is not accepted as refresh token", func(t *testing.T) {
		f := newHandlerFixture(t)
		pair := login(t, f)
		assertError(t, f.do(http.MethodPost, "/api/auth/refresh", "", `{"refresh_token":"`+pair.AccessToken+`"}`),
			http.StatusUnauthorized, "invalid credentials")
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
			assertError(t, f.do(http.MethodPost, "/api/auth/refresh", "", tt.body), tt.wantStatus, tt.wantMessage)
		})
	}
}

func TestHandlerMe(t *testing.T) {
	t.Run("returns the authenticated user", func(t *testing.T) {
		f := newHandlerFixture(t)
		u := f.registerAlice(t)
		pair, err := f.tokens.Issue(u.ID, u.Email, u.Roles)
		if err != nil {
			t.Fatal(err)
		}
		rec := f.do(http.MethodGet, "/api/me", pair.AccessToken, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		if got := decodeUser(t, rec); got.ID != u.ID || got.Email != u.Email {
			t.Errorf("got %+v; want %+v", got, u)
		}
	})

	t.Run("missing token", func(t *testing.T) {
		f := newHandlerFixture(t)
		assertError(t, f.do(http.MethodGet, "/api/me", "", ""), http.StatusUnauthorized, "missing credentials")
	})

	t.Run("valid token for deleted user", func(t *testing.T) {
		f := newHandlerFixture(t)
		pair, err := f.tokens.Issue(uuid.New(), "ghost@example.com", []string{model.RoleViewer})
		if err != nil {
			t.Fatal(err)
		}
		assertError(t, f.do(http.MethodGet, "/api/me", pair.AccessToken, ""), http.StatusNotFound, "user not found")
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
