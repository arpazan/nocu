package monitor

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
	repo     *fakeRepo
	viewer   string
	operator string
	admin    string
	adminID  uuid.UUID
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	svc, repo, _ := newService()
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
		repo:     repo,
		viewer:   issue(uuid.New(), model.RoleViewer),
		operator: issue(uuid.New(), model.RoleOperator),
		admin:    issue(adminID, model.RoleAdmin),
		adminID:  adminID,
	}
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

func decodeMonitor(t *testing.T, rec *httptest.ResponseRecorder) model.Monitor {
	t.Helper()
	var m model.Monitor
	if err := json.Unmarshal(decode(t, rec).Data, &m); err != nil {
		t.Fatalf("decode monitor: %v", err)
	}
	return m
}

const validBody = `{"name":"  API  ","url":"https://example.com/health"}`

func (f *handlerFixture) seed(t *testing.T) model.Monitor {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/monitors", f.admin, validBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed: got %d body %s", rec.Code, rec.Body.String())
	}
	return decodeMonitor(t, rec)
}

func TestHandlerAuthorization(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t).ID.String()

	tests := []struct {
		name   string
		method string
		path   string
		token  string
		body   string
		want   int
	}{
		{"anonymous list", http.MethodGet, "/api/monitors", "", "", http.StatusUnauthorized},
		{"viewer list", http.MethodGet, "/api/monitors", f.viewer, "", http.StatusOK},
		{"viewer get", http.MethodGet, "/api/monitors/" + id, f.viewer, "", http.StatusOK},
		{"viewer heartbeats", http.MethodGet, "/api/monitors/" + id + "/heartbeats", f.viewer, "", http.StatusOK},
		{"viewer create", http.MethodPost, "/api/monitors", f.viewer, validBody, http.StatusForbidden},
		{"operator create", http.MethodPost, "/api/monitors", f.operator, validBody, http.StatusCreated},
		{"viewer update", http.MethodPatch, "/api/monitors/" + id, f.viewer, `{"name":"y"}`, http.StatusForbidden},
		{"operator update", http.MethodPatch, "/api/monitors/" + id, f.operator, `{"name":"y"}`, http.StatusOK},
		{"viewer pause", http.MethodPost, "/api/monitors/" + id + "/pause", f.viewer, "", http.StatusForbidden},
		{"operator pause", http.MethodPost, "/api/monitors/" + id + "/pause", f.operator, "", http.StatusOK},
		{"viewer resume", http.MethodPost, "/api/monitors/" + id + "/resume", f.viewer, "", http.StatusForbidden},
		{"operator resume", http.MethodPost, "/api/monitors/" + id + "/resume", f.operator, "", http.StatusOK},
		{"viewer delete", http.MethodDelete, "/api/monitors/" + id, f.viewer, "", http.StatusForbidden},
		{"operator delete", http.MethodDelete, "/api/monitors/" + id, f.operator, "", http.StatusForbidden},
		{"admin delete", http.MethodDelete, "/api/monitors/" + id, f.admin, "", http.StatusNoContent},
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
		check       func(t *testing.T, m model.Monitor)
	}{
		{
			name:       "valid with defaults",
			body:       validBody,
			wantStatus: http.StatusCreated,
			check: func(t *testing.T, m model.Monitor) {
				if m.Name != "API" {
					t.Errorf("name = %q; want trimmed", m.Name)
				}
				if !m.Active {
					t.Error("new monitor should be active")
				}
				if m.IntervalSeconds <= 0 || m.TimeoutSeconds <= 0 {
					t.Errorf("defaults not applied: %+v", m)
				}
			},
		},
		{
			name: "explicit fields and notification",
			body: `{"name":"n","url":"https://x.io","interval_seconds":120,"timeout_seconds":10,"retries":2,
			        "method":"POST","body_encoding":"json","body":"{}","headers":{"X-A":"1"},
			        "notifications":[{"type":"webhook","enabled":true,"config":{"post_url":"https://hook.example"}}]}`,
			wantStatus: http.StatusCreated,
			check: func(t *testing.T, m model.Monitor) {
				if m.IntervalSeconds != 120 || m.TimeoutSeconds != 10 || m.Retries != 2 {
					t.Errorf("explicit fields lost: %+v", m)
				}
				if len(m.Notifications) != 1 || m.Notifications[0].Type != model.NotificationWebhook {
					t.Errorf("notifications = %+v", m.Notifications)
				}
			},
		},
		{
			name:        "malformed JSON",
			body:        `{"name":`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "invalid request body",
		},
		{
			name:        "missing name",
			body:        `{"url":"https://example.com"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "name is required",
		},
		{
			name:        "missing url",
			body:        `{"name":"x"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "url is required",
		},
		{
			name:        "non-http url",
			body:        `{"name":"x","url":"ftp://example.com"}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "url must be an http or https URL",
		},
		{
			name:        "timeout not below interval",
			body:        `{"name":"x","url":"https://example.com","interval_seconds":60,"timeout_seconds":60}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "timeout_seconds must be less than interval_seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			rec := f.do(http.MethodPost, "/api/monitors", f.admin, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantMessage != "" {
				if env := decode(t, rec); env.Error == nil || env.Error.Message != tt.wantMessage {
					t.Errorf("error = %+v; want message %q", env.Error, tt.wantMessage)
				}
				return
			}
			m := decodeMonitor(t, rec)
			if m.OwnerID == nil || *m.OwnerID != f.adminID {
				t.Errorf("owner_id = %v; want caller %s", m.OwnerID, f.adminID)
			}
			tt.check(t, m)
		})
	}
}

func TestHandlerList(t *testing.T) {
	t.Run("empty returns array with default meta", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/monitors", f.viewer, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		env := decode(t, rec)
		if string(env.Data) != "[]" {
			t.Errorf("data = %s; want []", env.Data)
		}
		if env.Meta == nil || env.Meta.Total != 0 || env.Meta.Limit != 50 || env.Meta.Offset != 0 {
			t.Errorf("meta = %+v; want total=0 limit=50 offset=0", env.Meta)
		}
	})

	t.Run("pagination normalised", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.seed(t)
		rec := f.do(http.MethodGet, "/api/monitors?limit=0&offset=-1", f.viewer, "")
		env := decode(t, rec)
		if env.Meta.Total != 1 || env.Meta.Limit != 50 || env.Meta.Offset != 0 {
			t.Errorf("meta = %+v", env.Meta)
		}
	})

	tests := []struct {
		name   string
		active string
		want   int
	}{
		{"active=true", "true", http.StatusOK},
		{"active=false", "false", http.StatusOK},
		{"active=1", "1", http.StatusOK},
		{"active=yes rejected", "yes", http.StatusBadRequest},
		{"active=maybe rejected", "maybe", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			rec := f.do(http.MethodGet, "/api/monitors?active="+tt.active, f.viewer, "")
			if rec.Code != tt.want {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.want, rec.Body.String())
			}
			if tt.want == http.StatusBadRequest {
				if env := decode(t, rec); env.Error.Message != "active must be true or false" {
					t.Errorf("message = %q", env.Error.Message)
				}
			}
		})
	}
}

func TestHandlerGet(t *testing.T) {
	f := newHandlerFixture(t)
	existing := f.seed(t)

	tests := []struct {
		name        string
		id          string
		wantStatus  int
		wantMessage string
	}{
		{"found", existing.ID.String(), http.StatusOK, ""},
		{"unknown id", uuid.New().String(), http.StatusNotFound, "monitor not found"},
		{"malformed id", "nope", http.StatusBadRequest, "id must be a UUID"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.do(http.MethodGet, "/api/monitors/"+tt.id, f.viewer, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantMessage != "" {
				if env := decode(t, rec); env.Error.Message != tt.wantMessage {
					t.Errorf("message = %q; want %q", env.Error.Message, tt.wantMessage)
				}
				return
			}
			if m := decodeMonitor(t, rec); m.ID != existing.ID {
				t.Errorf("id = %s; want %s", m.ID, existing.ID)
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
		check       func(t *testing.T, before, after model.Monitor)
	}{
		{
			name:       "rename only",
			body:       `{"name":"  Renamed "}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, before, after model.Monitor) {
				if after.Name != "Renamed" {
					t.Errorf("name = %q", after.Name)
				}
				if after.URL != before.URL || after.IntervalSeconds != before.IntervalSeconds {
					t.Errorf("untouched fields changed: before %+v after %+v", before, after)
				}
			},
		},
		{
			name:       "replace notifications",
			body:       `{"notifications":[{"type":"webhook","config":{"post_url":"https://hook.example"}}]}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, _, after model.Monitor) {
				if len(after.Notifications) != 1 {
					t.Errorf("notifications = %+v; want one", after.Notifications)
				}
			},
		},
		{
			name:       "clear notifications with empty array",
			body:       `{"notifications":[]}`,
			wantStatus: http.StatusOK,
			check: func(t *testing.T, _, after model.Monitor) {
				if len(after.Notifications) != 0 {
					t.Errorf("notifications = %+v; want none", after.Notifications)
				}
			},
		},
		{
			name:        "empty name",
			body:        `{"name":"  "}`,
			wantStatus:  http.StatusBadRequest,
			wantMessage: "name is required",
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
			before := f.seed(t)
			rec := f.do(http.MethodPatch, "/api/monitors/"+before.ID.String(), f.operator, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantMessage != "" {
				if env := decode(t, rec); env.Error.Message != tt.wantMessage {
					t.Errorf("message = %q; want %q", env.Error.Message, tt.wantMessage)
				}
				return
			}
			tt.check(t, before, decodeMonitor(t, rec))
		})
	}

	t.Run("unknown id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodPatch, "/api/monitors/"+uuid.New().String(), f.operator, `{"name":"x"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("got %d; want 404", rec.Code)
		}
	})
}

func TestHandlerPauseResume(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t).ID.String()

	rec := f.do(http.MethodPost, "/api/monitors/"+id+"/pause", f.operator, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("pause: got %d body %s", rec.Code, rec.Body.String())
	}
	if m := decodeMonitor(t, rec); m.Active {
		t.Error("pause should set active=false")
	}

	rec = f.do(http.MethodPost, "/api/monitors/"+id+"/resume", f.operator, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("resume: got %d body %s", rec.Code, rec.Body.String())
	}
	if m := decodeMonitor(t, rec); !m.Active {
		t.Error("resume should set active=true")
	}

	for _, action := range []string{"pause", "resume"} {
		if rec := f.do(http.MethodPost, "/api/monitors/"+uuid.New().String()+"/"+action, f.operator, ""); rec.Code != http.StatusNotFound {
			t.Errorf("%s unknown: got %d; want 404", action, rec.Code)
		}
		if rec := f.do(http.MethodPost, "/api/monitors/bad/"+action, f.operator, ""); rec.Code != http.StatusBadRequest {
			t.Errorf("%s malformed: got %d; want 400", action, rec.Code)
		}
	}
}

func TestHandlerHeartbeats(t *testing.T) {
	f := newHandlerFixture(t)
	m := f.seed(t)
	f.repo.heartbeats[m.ID] = []model.MonitorHeartbeat{
		{MonitorID: m.ID, Status: model.MonitorUp},
		{MonitorID: m.ID, Status: model.MonitorDown},
	}

	t.Run("lists with default meta", func(t *testing.T) {
		rec := f.do(http.MethodGet, "/api/monitors/"+m.ID.String()+"/heartbeats", f.viewer, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d body %s", rec.Code, rec.Body.String())
		}
		env := decode(t, rec)
		var items []model.MonitorHeartbeat
		if err := json.Unmarshal(env.Data, &items); err != nil || len(items) != 2 {
			t.Errorf("items = %s (err %v); want 2", env.Data, err)
		}
		if env.Meta == nil || env.Meta.Total != 2 || env.Meta.Limit != 100 || env.Meta.Offset != 0 {
			t.Errorf("meta = %+v; want total=2 limit=100 offset=0", env.Meta)
		}
	})

	t.Run("limit capped", func(t *testing.T) {
		rec := f.do(http.MethodGet, "/api/monitors/"+m.ID.String()+"/heartbeats?limit=5000&important=true", f.viewer, "")
		if env := decode(t, rec); env.Meta.Limit != 100 {
			t.Errorf("limit = %d; want 100", env.Meta.Limit)
		}
	})

	t.Run("empty returns array", func(t *testing.T) {
		other := f.seed(t)
		rec := f.do(http.MethodGet, "/api/monitors/"+other.ID.String()+"/heartbeats", f.viewer, "")
		if env := decode(t, rec); string(env.Data) != "[]" {
			t.Errorf("data = %s; want []", env.Data)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		rec := f.do(http.MethodGet, "/api/monitors/zzz/heartbeats", f.viewer, "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("got %d; want 400", rec.Code)
		}
	})
}

func TestHandlerDelete(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t).ID.String()

	if rec := f.do(http.MethodDelete, "/api/monitors/bad", f.admin, ""); rec.Code != http.StatusBadRequest {
		t.Errorf("malformed id: got %d; want 400", rec.Code)
	}
	if rec := f.do(http.MethodDelete, "/api/monitors/"+id, f.admin, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d body %s", rec.Code, rec.Body.String())
	}
	if rec := f.do(http.MethodGet, "/api/monitors/"+id, f.admin, ""); rec.Code != http.StatusNotFound {
		t.Errorf("after delete: got %d; want 404", rec.Code)
	}
	if rec := f.do(http.MethodDelete, "/api/monitors/"+id, f.admin, ""); rec.Code != http.StatusNotFound {
		t.Errorf("double delete: got %d; want 404", rec.Code)
	}
}
