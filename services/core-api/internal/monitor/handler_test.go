package monitor

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/model"
	"github.com/Pzquzer/Nocu/shared/testutil"
)

// handlerFixture wires the handler behind real JWT auth. Each test (or
// mutating subtest) builds its own so state never leaks between cases.
type handlerFixture struct {
	e       *echo.Echo
	repo    *fakeRepo // reached into only for heartbeat seeding, which has no write endpoint
	roles   map[string]string
	adminID uuid.UUID
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	svc, repo, _ := newService()
	tokens := testutil.NewTokenManager()

	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler(slog.Default())
	NewHandler(svc).Register(e.Group("/api", httpx.JWTAuth(tokens)))

	adminID := uuid.New()
	return &handlerFixture{
		e:    e,
		repo: repo,
		roles: map[string]string{
			model.RoleViewer:   testutil.IssueToken(t, tokens, uuid.New(), model.RoleViewer),
			model.RoleOperator: testutil.IssueToken(t, tokens, uuid.New(), model.RoleOperator),
			model.RoleAdmin:    testutil.IssueToken(t, tokens, adminID, model.RoleAdmin),
		},
		adminID: adminID,
	}
}

func (f *handlerFixture) token(role string) string { return f.roles[role] }

func (f *handlerFixture) do(method, path, role, body string) *httptest.ResponseRecorder {
	return testutil.Do(f.e, method, path, f.token(role), body)
}

const validBody = `{"name":"  API  ","url":"https://example.com/health"}`

func (f *handlerFixture) seed(t *testing.T) model.Monitor {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/monitors", model.RoleAdmin, validBody)
	return testutil.DecodeData[model.Monitor](t, rec, http.StatusCreated)
}

func TestHandlerAuthorization(t *testing.T) {
	// {id} is replaced with a freshly seeded monitor for every row, so rows
	// are independent and can run in any order or subset.
	tests := []struct {
		name   string
		method string
		path   string
		role   string
		body   string
		want   int
	}{
		{"anonymous list", http.MethodGet, "/api/monitors", "", "", http.StatusUnauthorized},
		{"viewer list", http.MethodGet, "/api/monitors", model.RoleViewer, "", http.StatusOK},
		{"viewer get", http.MethodGet, "/api/monitors/{id}", model.RoleViewer, "", http.StatusOK},
		{"viewer heartbeats", http.MethodGet, "/api/monitors/{id}/heartbeats", model.RoleViewer, "", http.StatusOK},
		{"viewer create", http.MethodPost, "/api/monitors", model.RoleViewer, validBody, http.StatusForbidden},
		{"operator create", http.MethodPost, "/api/monitors", model.RoleOperator, validBody, http.StatusCreated},
		{"viewer update", http.MethodPatch, "/api/monitors/{id}", model.RoleViewer, `{"name":"y"}`, http.StatusForbidden},
		{"operator update", http.MethodPatch, "/api/monitors/{id}", model.RoleOperator, `{"name":"y"}`, http.StatusOK},
		{"viewer pause", http.MethodPost, "/api/monitors/{id}/pause", model.RoleViewer, "", http.StatusForbidden},
		{"operator pause", http.MethodPost, "/api/monitors/{id}/pause", model.RoleOperator, "", http.StatusOK},
		{"viewer resume", http.MethodPost, "/api/monitors/{id}/resume", model.RoleViewer, "", http.StatusForbidden},
		{"operator resume", http.MethodPost, "/api/monitors/{id}/resume", model.RoleOperator, "", http.StatusOK},
		{"viewer delete", http.MethodDelete, "/api/monitors/{id}", model.RoleViewer, "", http.StatusForbidden},
		{"operator delete", http.MethodDelete, "/api/monitors/{id}", model.RoleOperator, "", http.StatusForbidden},
		{"admin delete", http.MethodDelete, "/api/monitors/{id}", model.RoleAdmin, "", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			id := f.seed(t).ID.String()
			rec := f.do(tt.method, strings.ReplaceAll(tt.path, "{id}", id), tt.role, tt.body)
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
			rec := f.do(http.MethodPost, "/api/monitors", model.RoleAdmin, tt.body)
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			m := testutil.DecodeData[model.Monitor](t, rec, tt.wantStatus)
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
		data, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/monitors", model.RoleViewer, ""))
		if string(data) != "[]" {
			t.Errorf("data = %s; want []", data)
		}
		if meta != (httpx.Meta{Total: 0, Limit: 50, Offset: 0}) {
			t.Errorf("meta = %+v; want total=0 limit=50 offset=0", meta)
		}
	})

	t.Run("pagination normalised", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.seed(t)
		_, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/monitors?limit=0&offset=-1", model.RoleViewer, ""))
		if meta != (httpx.Meta{Total: 1, Limit: 50, Offset: 0}) {
			t.Errorf("meta = %+v; want total=1 limit=50 offset=0", meta)
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
			rec := f.do(http.MethodGet, "/api/monitors?active="+tt.active, model.RoleViewer, "")
			if tt.want == http.StatusBadRequest {
				testutil.AssertError(t, rec, tt.want, "active must be true or false")
				return
			}
			if rec.Code != tt.want {
				t.Fatalf("got %d; want %d (body %s)", rec.Code, tt.want, rec.Body.String())
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

	// Read-only rows: sharing one seeded fixture is safe here.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.do(http.MethodGet, "/api/monitors/"+tt.id, model.RoleViewer, "")
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			if m := testutil.DecodeData[model.Monitor](t, rec, tt.wantStatus); m.ID != existing.ID {
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
			rec := f.do(http.MethodPatch, "/api/monitors/"+before.ID.String(), model.RoleOperator, tt.body)
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			tt.check(t, before, testutil.DecodeData[model.Monitor](t, rec, tt.wantStatus))
		})
	}

	t.Run("unknown id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodPatch, "/api/monitors/"+uuid.New().String(), model.RoleOperator, `{"name":"x"}`)
		testutil.AssertError(t, rec, http.StatusNotFound, "monitor not found")
	})
}

func TestHandlerPauseResume(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t).ID.String()

	paused := testutil.DecodeData[model.Monitor](t, f.do(http.MethodPost, "/api/monitors/"+id+"/pause", model.RoleOperator, ""), http.StatusOK)
	if paused.Active {
		t.Error("pause should set active=false")
	}
	resumed := testutil.DecodeData[model.Monitor](t, f.do(http.MethodPost, "/api/monitors/"+id+"/resume", model.RoleOperator, ""), http.StatusOK)
	if !resumed.Active {
		t.Error("resume should set active=true")
	}

	for _, action := range []string{"pause", "resume"} {
		t.Run(action+" unknown id", func(t *testing.T) {
			rec := f.do(http.MethodPost, "/api/monitors/"+uuid.New().String()+"/"+action, model.RoleOperator, "")
			testutil.AssertError(t, rec, http.StatusNotFound, "monitor not found")
		})
		t.Run(action+" malformed id", func(t *testing.T) {
			rec := f.do(http.MethodPost, "/api/monitors/bad/"+action, model.RoleOperator, "")
			testutil.AssertError(t, rec, http.StatusBadRequest, "id must be a UUID")
		})
	}
}

func TestHandlerHeartbeats(t *testing.T) {
	f := newHandlerFixture(t)
	m := f.seed(t)
	// Heartbeats are written by the scheduler, not over HTTP, so seed the
	// fake repository directly.
	f.repo.heartbeats[m.ID] = []model.MonitorHeartbeat{
		{MonitorID: m.ID, Status: model.MonitorUp},
		{MonitorID: m.ID, Status: model.MonitorDown},
	}

	t.Run("lists with default meta", func(t *testing.T) {
		data, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/monitors/"+m.ID.String()+"/heartbeats", model.RoleViewer, ""))
		var items []model.MonitorHeartbeat
		if err := json.Unmarshal(data, &items); err != nil || len(items) != 2 {
			t.Errorf("items = %s (err %v); want 2", data, err)
		}
		if meta != (httpx.Meta{Total: 2, Limit: 100, Offset: 0}) {
			t.Errorf("meta = %+v; want total=2 limit=100 offset=0", meta)
		}
	})

	t.Run("limit capped", func(t *testing.T) {
		_, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/monitors/"+m.ID.String()+"/heartbeats?limit=5000&important=true", model.RoleViewer, ""))
		if meta.Limit != 100 {
			t.Errorf("limit = %d; want 100", meta.Limit)
		}
	})

	t.Run("empty returns array", func(t *testing.T) {
		other := f.seed(t)
		data, _ := testutil.DecodeList(t, f.do(http.MethodGet, "/api/monitors/"+other.ID.String()+"/heartbeats", model.RoleViewer, ""))
		if string(data) != "[]" {
			t.Errorf("data = %s; want []", data)
		}
	})

	t.Run("malformed id", func(t *testing.T) {
		rec := f.do(http.MethodGet, "/api/monitors/zzz/heartbeats", model.RoleViewer, "")
		testutil.AssertError(t, rec, http.StatusBadRequest, "id must be a UUID")
	})
}

func TestHandlerDelete(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t).ID.String()

	testutil.AssertError(t, f.do(http.MethodDelete, "/api/monitors/bad", model.RoleAdmin, ""), http.StatusBadRequest, "id must be a UUID")
	if rec := f.do(http.MethodDelete, "/api/monitors/"+id, model.RoleAdmin, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d body %s", rec.Code, rec.Body.String())
	}
	testutil.AssertError(t, f.do(http.MethodGet, "/api/monitors/"+id, model.RoleAdmin, ""), http.StatusNotFound, "monitor not found")
	testutil.AssertError(t, f.do(http.MethodDelete, "/api/monitors/"+id, model.RoleAdmin, ""), http.StatusNotFound, "monitor not found")
}
