package incident

import (
	"encoding/json"
	"log/slog"
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
	"github.com/Pzquzer/Nocu/shared/model"
)

type handlerFixture struct {
	e        *echo.Echo
	svc      *Service
	tokens   *auth.Manager
	viewer   string
	operator string
	admin    string
	adminID  uuid.UUID
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	svc, _ := newService()
	tokens := auth.NewManager(config.JWT{Secret: "unit-test-secret-value", Issuer: "t", AccessTTL: time.Minute, RefreshTTL: time.Hour})

	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler(slog.Default())
	g := e.Group("/api", httpx.JWTAuth(tokens))
	NewHandler(svc).Register(g)

	issue := func(id uuid.UUID, role string) string {
		pair, err := tokens.Issue(id, role+"@example.com", []string{role})
		if err != nil {
			t.Fatalf("issue %s token: %v", role, err)
		}
		return pair.AccessToken
	}
	adminID := uuid.New()
	return &handlerFixture{
		e:        e,
		svc:      svc,
		tokens:   tokens,
		viewer:   issue(uuid.New(), model.RoleViewer),
		operator: issue(uuid.New(), model.RoleOperator),
		admin:    issue(adminID, model.RoleAdmin),
		adminID:  adminID,
	}
}

func (f *handlerFixture) do(method, path, token, body string) *httptest.ResponseRecorder {
	var rd *strings.Reader
	if body != "" {
		rd = strings.NewReader(body)
	} else {
		rd = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, rd)
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
	Meta  *httpx.Meta     `json:"meta"`
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

func decodeIncident(t *testing.T, rec *httptest.ResponseRecorder) model.Incident {
	t.Helper()
	var inc model.Incident
	if err := json.Unmarshal(decode(t, rec).Data, &inc); err != nil {
		t.Fatalf("decode incident: %v", err)
	}
	return inc
}

func (f *handlerFixture) seed(t *testing.T, title string) model.Incident {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/incidents", f.admin, `{"title":"`+title+`","severity":"high"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed: got %d body %s", rec.Code, rec.Body.String())
	}
	return decodeIncident(t, rec)
}

func TestHandlerAuthorization(t *testing.T) {
	f := newHandlerFixture(t)
	existing := f.seed(t, "seed")
	id := existing.ID.String()

	tests := []struct {
		name   string
		method string
		path   string
		token  string
		body   string
		want   int
	}{
		{"anonymous list", http.MethodGet, "/api/incidents", "", "", http.StatusUnauthorized},
		{"viewer list", http.MethodGet, "/api/incidents", f.viewer, "", http.StatusOK},
		{"viewer get", http.MethodGet, "/api/incidents/" + id, f.viewer, "", http.StatusOK},
		{"viewer create", http.MethodPost, "/api/incidents", f.viewer, `{"title":"x","severity":"low"}`, http.StatusForbidden},
		{"operator create", http.MethodPost, "/api/incidents", f.operator, `{"title":"x","severity":"low"}`, http.StatusCreated},
		{"viewer update", http.MethodPatch, "/api/incidents/" + id, f.viewer, `{"title":"y"}`, http.StatusForbidden},
		{"operator update", http.MethodPatch, "/api/incidents/" + id, f.operator, `{"title":"y"}`, http.StatusOK},
		{"viewer delete", http.MethodDelete, "/api/incidents/" + id, f.viewer, "", http.StatusForbidden},
		{"operator delete", http.MethodDelete, "/api/incidents/" + id, f.operator, "", http.StatusForbidden},
		{"admin delete", http.MethodDelete, "/api/incidents/" + id, f.admin, "", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.do(tt.method, tt.path, tt.token, tt.body)
			if rec.Code != tt.want {
				t.Errorf("got %d; want %d (body %s)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}

func TestHandlerCreate(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
	}{
		{
			name:       "valid",
			body:       `{"title":"  Router down ","description":" core ","severity":"critical","source":"manual"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:        "malformed JSON",
			body:        `{"title":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request body",
		},
		{
			name:        "missing title",
			body:        `{"severity":"high"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "title is required",
		},
		{
			name:        "unknown severity",
			body:        `{"title":"x","severity":"meh"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "severity must be one of critical, high, medium, low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			rec := f.do(http.MethodPost, "/api/incidents", f.admin, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				env := decode(t, rec)
				if env.Error == nil || env.Error.Message != tt.wantMessage {
					t.Errorf("error = %+v; want message %q", env.Error, tt.wantMessage)
				}
				return
			}
			inc := decodeIncident(t, rec)
			if inc.Title != "Router down" || inc.Description != "core" || inc.Source != "manual" {
				t.Errorf("fields not trimmed: %+v", inc)
			}
			if inc.Status != model.IncidentOpen {
				t.Errorf("status = %q; want open", inc.Status)
			}
			if inc.ReporterID == nil || *inc.ReporterID != f.adminID {
				t.Errorf("reporter_id = %v; want caller %s", inc.ReporterID, f.adminID)
			}
		})
	}
}

func TestHandlerList(t *testing.T) {
	t.Run("empty returns array not null", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents", f.viewer, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		env := decode(t, rec)
		if string(env.Data) != "[]" {
			t.Errorf("data = %s; want []", env.Data)
		}
		if env.Meta == nil || env.Meta.Total != 0 || env.Meta.Limit != 50 || env.Meta.Offset != 0 {
			t.Errorf("meta = %+v; want defaults total=0 limit=50 offset=0", env.Meta)
		}
	})

	t.Run("pagination normalised", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.seed(t, "a")
		f.seed(t, "b")
		rec := f.do(http.MethodGet, "/api/incidents?limit=999&offset=-3", f.viewer, "")
		env := decode(t, rec)
		if env.Meta.Total != 2 || env.Meta.Limit != 50 || env.Meta.Offset != 0 {
			t.Errorf("meta = %+v; want total=2 limit=50 offset=0", env.Meta)
		}
	})

	t.Run("explicit pagination honoured", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents?limit=10&offset=20", f.viewer, "")
		env := decode(t, rec)
		if env.Meta.Limit != 10 || env.Meta.Offset != 20 {
			t.Errorf("meta = %+v; want limit=10 offset=20", env.Meta)
		}
	})

	t.Run("invalid monitor_id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents?monitor_id=not-a-uuid", f.viewer, "")
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("got %d; want 400", rec.Code)
		}
		if env := decode(t, rec); env.Error.Message != "monitor_id must be a UUID" {
			t.Errorf("message = %q", env.Error.Message)
		}
	})

	t.Run("valid monitor_id accepted", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents?monitor_id="+uuid.New().String(), f.viewer, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
	})
}

func TestHandlerGet(t *testing.T) {
	f := newHandlerFixture(t)
	existing := f.seed(t, "seed")

	tests := []struct {
		name        string
		id          string
		wantStatus  int
		wantMessage string
	}{
		{"found", existing.ID.String(), http.StatusOK, ""},
		{"unknown id", uuid.New().String(), http.StatusNotFound, "incident not found"},
		{"malformed id", "abc", http.StatusBadRequest, "id must be a UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.do(http.MethodGet, "/api/incidents/"+tt.id, f.viewer, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantMessage != "" {
				if env := decode(t, rec); env.Error.Message != tt.wantMessage {
					t.Errorf("message = %q; want %q", env.Error.Message, tt.wantMessage)
				}
				return
			}
			if inc := decodeIncident(t, rec); inc.ID != existing.ID {
				t.Errorf("id = %s; want %s", inc.ID, existing.ID)
			}
		})
	}
}

func TestHandlerUpdate(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantMessage string
		check       func(t *testing.T, inc model.Incident)
	}{
		{
			name:       "acknowledge",
			body:       `{"status":"acknowledged"}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, inc model.Incident) {
				if inc.Status != model.IncidentAcknowledged || inc.AcknowledgedAt == nil {
					t.Errorf("expected acknowledged with timestamp: %+v", inc)
				}
			},
		},
		{
			name:       "rename",
			body:       `{"title":"  New title "}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, inc model.Incident) {
				if inc.Title != "New title" {
					t.Errorf("title = %q", inc.Title)
				}
			},
		},
		{
			name:        "unknown status",
			body:        `{"status":"bogus"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "unknown status",
		},
		{
			name:        "empty title",
			body:        `{"title":"   "}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "title cannot be empty",
		},
		{
			name:        "malformed JSON",
			body:        `{`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			existing := f.seed(t, "seed")
			rec := f.do(http.MethodPatch, "/api/incidents/"+existing.ID.String(), f.operator, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantMessage != "" {
				if env := decode(t, rec); env.Error.Message != tt.wantMessage {
					t.Errorf("message = %q; want %q", env.Error.Message, tt.wantMessage)
				}
				return
			}
			tt.check(t, decodeIncident(t, rec))
		})
	}

	t.Run("reopening a closed incident is a conflict", func(t *testing.T) {
		f := newHandlerFixture(t)
		existing := f.seed(t, "seed")
		path := "/api/incidents/" + existing.ID.String()
		if rec := f.do(http.MethodPatch, path, f.operator, `{"status":"closed"}`); rec.Code != http.StatusOK {
			t.Fatalf("close: got %d body %s", rec.Code, rec.Body.String())
		}
		rec := f.do(http.MethodPatch, path, f.operator, `{"status":"open"}`)
		if rec.Code != http.StatusConflict {
			t.Fatalf("reopen: got %d; want 409 (body %s)", rec.Code, rec.Body.String())
		}
		if env := decode(t, rec); env.Error.Message != "invalid status transition: closed -> open" {
			t.Errorf("message = %q", env.Error.Message)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodPatch, "/api/incidents/"+uuid.New().String(), f.operator, `{"title":"x"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("got %d; want 404", rec.Code)
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	f := newHandlerFixture(t)
	existing := f.seed(t, "seed")
	id := existing.ID.String()

	if rec := f.do(http.MethodDelete, "/api/incidents/bad", f.admin, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed id: got %d; want 400", rec.Code)
	}
	if rec := f.do(http.MethodDelete, "/api/incidents/"+id, f.admin, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d body %s", rec.Code, rec.Body.String())
	}
	if rec := f.do(http.MethodGet, "/api/incidents/"+id, f.admin, ""); rec.Code != http.StatusNotFound {
		t.Errorf("after delete: got %d; want 404", rec.Code)
	}
	if rec := f.do(http.MethodDelete, "/api/incidents/"+id, f.admin, ""); rec.Code != http.StatusNotFound {
		t.Errorf("double delete: got %d; want 404", rec.Code)
	}
}
