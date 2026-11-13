package incident

import (
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
	roles   map[string]string // role name -> access token
	adminID uuid.UUID
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	svc, _ := newService()
	tokens := testutil.NewTokenManager()

	e := echo.New()
	e.HTTPErrorHandler = httpx.ErrorHandler(slog.Default())
	NewHandler(svc).Register(e.Group("/api", httpx.JWTAuth(tokens)))

	adminID := uuid.New()
	return &handlerFixture{
		e: e,
		roles: map[string]string{
			model.RoleViewer:   testutil.IssueToken(t, tokens, uuid.New(), model.RoleViewer),
			model.RoleOperator: testutil.IssueToken(t, tokens, uuid.New(), model.RoleOperator),
			model.RoleAdmin:    testutil.IssueToken(t, tokens, adminID, model.RoleAdmin),
		},
		adminID: adminID,
	}
}

// token returns the access token for role, or "" for an anonymous request.
func (f *handlerFixture) token(role string) string { return f.roles[role] }

func (f *handlerFixture) do(method, path, role, body string) *httptest.ResponseRecorder {
	return testutil.Do(f.e, method, path, f.token(role), body)
}

func (f *handlerFixture) seed(t *testing.T, title string) model.Incident {
	t.Helper()
	rec := f.do(http.MethodPost, "/api/incidents", model.RoleAdmin, `{"title":"`+title+`","severity":"high"}`)
	return testutil.DecodeData[model.Incident](t, rec, http.StatusCreated)
}

func TestHandlerAuthorization(t *testing.T) {
	// {id} is replaced with a freshly seeded incident for every row, so rows
	// are independent and can run in any order or subset.
	tests := []struct {
		name   string
		method string
		path   string
		role   string
		body   string
		want   int
	}{
		{"anonymous list", http.MethodGet, "/api/incidents", "", "", http.StatusUnauthorized},
		{"viewer list", http.MethodGet, "/api/incidents", model.RoleViewer, "", http.StatusOK},
		{"viewer get", http.MethodGet, "/api/incidents/{id}", model.RoleViewer, "", http.StatusOK},
		{"viewer create", http.MethodPost, "/api/incidents", model.RoleViewer, `{"title":"x","severity":"low"}`, http.StatusForbidden},
		{"operator create", http.MethodPost, "/api/incidents", model.RoleOperator, `{"title":"x","severity":"low"}`, http.StatusCreated},
		{"viewer update", http.MethodPatch, "/api/incidents/{id}", model.RoleViewer, `{"title":"y"}`, http.StatusForbidden},
		{"operator update", http.MethodPatch, "/api/incidents/{id}", model.RoleOperator, `{"title":"y"}`, http.StatusOK},
		{"viewer delete", http.MethodDelete, "/api/incidents/{id}", model.RoleViewer, "", http.StatusForbidden},
		{"operator delete", http.MethodDelete, "/api/incidents/{id}", model.RoleOperator, "", http.StatusForbidden},
		{"admin delete", http.MethodDelete, "/api/incidents/{id}", model.RoleAdmin, "", http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newHandlerFixture(t)
			id := f.seed(t, "seed").ID.String()
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
			rec := f.do(http.MethodPost, "/api/incidents", model.RoleAdmin, tt.body)
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			inc := testutil.DecodeData[model.Incident](t, rec, tt.wantStatus)
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
		data, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/incidents", model.RoleViewer, ""))
		if string(data) != "[]" {
			t.Errorf("data = %s; want []", data)
		}
		if meta != (httpx.Meta{Total: 0, Limit: 50, Offset: 0}) {
			t.Errorf("meta = %+v; want defaults total=0 limit=50 offset=0", meta)
		}
	})

	t.Run("pagination normalised", func(t *testing.T) {
		f := newHandlerFixture(t)
		f.seed(t, "a")
		f.seed(t, "b")
		_, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/incidents?limit=999&offset=-3", model.RoleViewer, ""))
		if meta != (httpx.Meta{Total: 2, Limit: 50, Offset: 0}) {
			t.Errorf("meta = %+v; want total=2 limit=50 offset=0", meta)
		}
	})

	t.Run("explicit pagination honoured", func(t *testing.T) {
		f := newHandlerFixture(t)
		_, meta := testutil.DecodeList(t, f.do(http.MethodGet, "/api/incidents?limit=10&offset=20", model.RoleViewer, ""))
		if meta.Limit != 10 || meta.Offset != 20 {
			t.Errorf("meta = %+v; want limit=10 offset=20", meta)
		}
	})

	t.Run("invalid monitor_id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents?monitor_id=not-a-uuid", model.RoleViewer, "")
		testutil.AssertError(t, rec, http.StatusBadRequest, "monitor_id must be a UUID")
	})

	t.Run("valid monitor_id accepted", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodGet, "/api/incidents?monitor_id="+uuid.New().String(), model.RoleViewer, "")
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

	// Read-only rows: sharing one seeded fixture is safe here.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.do(http.MethodGet, "/api/incidents/"+tt.id, model.RoleViewer, "")
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			if inc := testutil.DecodeData[model.Incident](t, rec, tt.wantStatus); inc.ID != existing.ID {
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
			rec := f.do(http.MethodPatch, "/api/incidents/"+existing.ID.String(), model.RoleOperator, tt.body)
			if tt.wantMessage != "" {
				testutil.AssertError(t, rec, tt.wantStatus, tt.wantMessage)
				return
			}
			tt.check(t, testutil.DecodeData[model.Incident](t, rec, tt.wantStatus))
		})
	}

	t.Run("reopening a closed incident is a conflict", func(t *testing.T) {
		f := newHandlerFixture(t)
		path := "/api/incidents/" + f.seed(t, "seed").ID.String()
		testutil.DecodeData[model.Incident](t, f.do(http.MethodPatch, path, model.RoleOperator, `{"status":"closed"}`), http.StatusOK)
		rec := f.do(http.MethodPatch, path, model.RoleOperator, `{"status":"open"}`)
		testutil.AssertError(t, rec, http.StatusConflict, "invalid status transition: closed -> open")
	})

	t.Run("unknown id", func(t *testing.T) {
		f := newHandlerFixture(t)
		rec := f.do(http.MethodPatch, "/api/incidents/"+uuid.New().String(), model.RoleOperator, `{"title":"x"}`)
		testutil.AssertError(t, rec, http.StatusNotFound, "incident not found")
	})
}

func TestHandlerDelete(t *testing.T) {
	f := newHandlerFixture(t)
	id := f.seed(t, "seed").ID.String()

	testutil.AssertError(t, f.do(http.MethodDelete, "/api/incidents/bad", model.RoleAdmin, ""), http.StatusBadRequest, "id must be a UUID")
	if rec := f.do(http.MethodDelete, "/api/incidents/"+id, model.RoleAdmin, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d body %s", rec.Code, rec.Body.String())
	}
	testutil.AssertError(t, f.do(http.MethodGet, "/api/incidents/"+id, model.RoleAdmin, ""), http.StatusNotFound, "incident not found")
	testutil.AssertError(t, f.do(http.MethodDelete, "/api/incidents/"+id, model.RoleAdmin, ""), http.StatusNotFound, "incident not found")
}
