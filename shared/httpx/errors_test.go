package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestErrorConstructors(t *testing.T) {
	tests := []struct {
		name       string
		build      func(string) *Error
		wantStatus int
		wantCode   string
	}{
		{"bad request", BadRequest, http.StatusBadRequest, "bad_request"},
		{"unauthorized", Unauthorized, http.StatusUnauthorized, "unauthorized"},
		{"forbidden", Forbidden, http.StatusForbidden, "forbidden"},
		{"not found", NotFound, http.StatusNotFound, "not_found"},
		{"conflict", Conflict, http.StatusConflict, "conflict"},
		{"internal", Internal, http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.build("boom")
			if e.Status != tt.wantStatus {
				t.Errorf("Status = %d; want %d", e.Status, tt.wantStatus)
			}
			if e.Code != tt.wantCode {
				t.Errorf("Code = %q; want %q", e.Code, tt.wantCode)
			}
			if e.Message != "boom" {
				t.Errorf("Message = %q; want %q", e.Message, "boom")
			}
			if got, want := e.Error(), tt.wantCode+": boom"; got != want {
				t.Errorf("Error() = %q; want %q", got, want)
			}
		})
	}
}

func TestErrorWithCause(t *testing.T) {
	cause := errors.New("db exploded")
	base := Internal("something went wrong")
	wrapped := base.WithCause(cause)

	if base.Unwrap() != nil {
		t.Error("WithCause mutated the original error")
	}
	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is should find the cause through Unwrap")
	}
	if wrapped.Status != base.Status || wrapped.Code != base.Code || wrapped.Message != base.Message {
		t.Errorf("WithCause changed public fields: %+v", wrapped)
	}

	var target *Error
	if !errors.As(wrapped, &target) {
		t.Error("errors.As should match *Error")
	}
}

func TestCodeFor(t *testing.T) {
	tests := []struct {
		status int
		want   string
	}{
		{http.StatusBadRequest, "bad_request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusForbidden, "forbidden"},
		{http.StatusNotFound, "not_found"},
		{http.StatusMethodNotAllowed, "method_not_allowed"},
		{http.StatusConflict, "conflict"},
		{http.StatusUnprocessableEntity, "unprocessable"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusInternalServerError, "internal_error"},
		{http.StatusBadGateway, "internal_error"},
		{http.StatusTeapot, "error"},
		{http.StatusGone, "error"},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			if got := codeFor(tt.status); got != tt.want {
				t.Errorf("codeFor(%d) = %q; want %q", tt.status, got, tt.want)
			}
		})
	}
}

func newErrorEcho(handler echo.HandlerFunc) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = ErrorHandler(slog.New(slog.NewTextHandler(discard{}, nil)))
	e.Any("/x", handler)
	return e
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "shaped error passes through",
			method:     http.MethodGet,
			err:        Conflict("already exists"),
			wantStatus: http.StatusConflict,
			wantBody:   `{"error":{"code":"conflict","message":"already exists"}}` + "\n",
		},
		{
			name:       "shaped error with cause hides cause",
			method:     http.MethodGet,
			err:        BadRequest("invalid request body").WithCause(errors.New("secret detail")),
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"error":{"code":"bad_request","message":"invalid request body"}}` + "\n",
		},
		{
			name:       "echo HTTPError with string message",
			method:     http.MethodGet,
			err:        echo.NewHTTPError(http.StatusUnprocessableEntity, "cannot process"),
			wantStatus: http.StatusUnprocessableEntity,
			wantBody:   `{"error":{"code":"unprocessable","message":"cannot process"}}` + "\n",
		},
		{
			name:       "echo HTTPError with non-string message falls back to status text",
			method:     http.MethodGet,
			err:        echo.NewHTTPError(http.StatusTooManyRequests, map[string]string{"k": "v"}),
			wantStatus: http.StatusTooManyRequests,
			wantBody:   `{"error":{"code":"rate_limited","message":"Too Many Requests"}}` + "\n",
		},
		{
			name:       "unknown error becomes 500 without leaking detail",
			method:     http.MethodGet,
			err:        errors.New("pq: connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantBody:   `{"error":{"code":"internal_error","message":"something went wrong"}}` + "\n",
		},
		{
			name:       "HEAD request gets no body",
			method:     http.MethodHead,
			err:        NotFound("nope"),
			wantStatus: http.StatusNotFound,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newErrorEcho(func(echo.Context) error { return tt.err })
			req := httptest.NewRequest(tt.method, "/x", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestErrorHandlerSkipsCommittedResponse(t *testing.T) {
	e := newErrorEcho(func(c echo.Context) error {
		_ = c.String(http.StatusAccepted, "partial")
		return errors.New("too late")
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d; want %d (handler must not overwrite committed response)", rec.Code, http.StatusAccepted)
	}
	if rec.Body.String() != "partial" {
		t.Errorf("body = %q; want %q", rec.Body.String(), "partial")
	}
}

func TestResponseHelpers(t *testing.T) {
	tests := []struct {
		name       string
		handler    echo.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name:       "OK",
			handler:    func(c echo.Context) error { return OK(c, map[string]int{"n": 1}) },
			wantStatus: http.StatusOK,
			wantBody:   `{"data":{"n":1}}` + "\n",
		},
		{
			name:       "Created",
			handler:    func(c echo.Context) error { return Created(c, "x") },
			wantStatus: http.StatusCreated,
			wantBody:   `{"data":"x"}` + "\n",
		},
		{
			name: "List includes meta",
			handler: func(c echo.Context) error {
				return List(c, []int{1, 2}, Meta{Total: 10, Limit: 2, Offset: 4})
			},
			wantStatus: http.StatusOK,
			wantBody:   `{"data":[1,2],"meta":{"total":10,"limit":2,"offset":4}}` + "\n",
		},
		{
			name:       "NoContent",
			handler:    NoContent,
			wantStatus: http.StatusNoContent,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newErrorEcho(tt.handler)
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
			if rec.Body.String() != tt.wantBody {
				t.Errorf("body = %q; want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestBind(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantName   string
	}{
		{"valid JSON", `{"name":"alice"}`, http.StatusOK, "alice"},
		{"malformed JSON", `{"name":`, http.StatusBadRequest, ""},
		{"wrong type", `{"name":123}`, http.StatusBadRequest, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got payload
			e := newErrorEcho(func(c echo.Context) error {
				if err := Bind(c, &got); err != nil {
					return err
				}
				return NoContent(c)
			})
			req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(tt.body))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if tt.wantStatus == http.StatusOK {
				if rec.Code != http.StatusNoContent {
					t.Fatalf("status = %d; want 204, body %s", rec.Code, rec.Body.String())
				}
				if got.Name != tt.wantName {
					t.Errorf("Name = %q; want %q", got.Name, tt.wantName)
				}
				return
			}
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d; want %d", rec.Code, tt.wantStatus)
			}
			if !strings.Contains(rec.Body.String(), `"code":"bad_request"`) {
				t.Errorf("body should carry bad_request code: %s", rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "invalid request body") {
				t.Errorf("body should carry generic message, not decoder detail: %s", rec.Body.String())
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"standard", "Bearer abc.def", "abc.def"},
		{"lowercase scheme", "bearer abc", "abc"},
		{"mixed case scheme", "BeArEr abc", "abc"},
		{"extra whitespace", "Bearer   abc  ", "abc"},
		{"empty header", "", ""},
		{"scheme only", "Bearer ", ""},
		{"wrong scheme", "Basic abc", ""},
		{"prefix without space", "Bearerabc", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bearerToken(tt.header); got != tt.want {
				t.Errorf("bearerToken(%q) = %q; want %q", tt.header, got, tt.want)
			}
		})
	}
}
