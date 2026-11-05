package uptime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Pzquzer/Nocu/shared/model"
)

const (
	userAgent    = "Nocu-Uptime/1.0"
	maxErrorBody = 64 << 10 // bytes of a failed response kept for notifications
	maxDrain     = 1 << 20  // bytes read before abandoning a body
)

// Result is the outcome of a single probe.
type Result struct {
	Status        model.MonitorStatus
	HTTPStatus    int
	Latency       time.Duration
	Message       string
	ErrorBody     string
	CertExpiresAt *time.Time
}

// Prober performs HTTP checks against a monitor's URL.
type Prober struct {
	allowPrivate bool
	resolver     Resolver
}

// NewProber builds a Prober. allowPrivate disables the SSRF guard.
func NewProber(allowPrivate bool) *Prober {
	return &Prober{allowPrivate: allowPrivate}
}

// WithResolver overrides DNS resolution (tests).
func (p *Prober) WithResolver(r Resolver) *Prober {
	p.resolver = r
	return p
}

// Probe runs one check. It never returns an error: every failure is a down Result with a message.
func (p *Prober) Probe(ctx context.Context, m *model.Monitor) Result {
	req, err := buildRequest(ctx, m)
	if err != nil {
		return Result{Status: model.MonitorDown, Message: err.Error()}
	}

	timeout := m.Timeout()
	if timeout <= 0 {
		timeout = time.Duration(model.MonitorDefaultTimeout) * time.Second
	}
	maxRedirects := m.MaxRedirects
	if maxRedirects < 0 {
		maxRedirects = 0
	}
	client := &http.Client{
		Transport: newTransport(p.allowPrivate, p.resolver),
		Timeout:   timeout,
		CheckRedirect: func(_ *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	start := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return Result{Status: model.MonitorDown, Latency: latency, Message: describe(err, timeout)}
	}
	defer resp.Body.Close()

	res := Result{HTTPStatus: resp.StatusCode, Latency: latency}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		exp := resp.TLS.PeerCertificates[0].NotAfter.UTC()
		res.CertExpiresAt = &exp
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		res.Status = model.MonitorUp
		res.Message = resp.Status
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
		return res
	}

	res.Status = model.MonitorDown
	res.Message = "unexpected status " + resp.Status
	if m.SaveErrorResponse {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		res.ErrorBody = string(body)
	} else {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxDrain))
	}
	return res
}

func buildRequest(ctx context.Context, m *model.Monitor) (*http.Request, error) {
	u, err := url.Parse(strings.TrimSpace(m.URL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("invalid url: must be http or https")
	}

	method := string(m.Method)
	if method == "" {
		method = http.MethodGet
	}
	var body io.Reader
	hasBody := m.Body != "" && methodAllowsBody(method)
	if hasBody {
		body = strings.NewReader(m.Body)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	headers, err := m.HeaderMap()
	if err != nil {
		return nil, fmt.Errorf("invalid headers: %w", err)
	}
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	if hasBody && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", m.BodyEncoding.ContentType())
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", userAgent)
	}

	switch m.AuthMethod {
	case model.AuthBasic:
		req.SetBasicAuth(m.BasicUsername, m.BasicPassword)
	case model.AuthBearer:
		req.Header.Set("Authorization", "Bearer "+m.BearerToken)
	}
	return req, nil
}

func methodAllowsBody(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	return true
}

// describe turns a transport error into a short operator-facing message.
func describe(err error, timeout time.Duration) string {
	if errors.Is(err, ErrPrivateAddress) {
		return "blocked: target resolves to a private or local address"
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() || errors.Is(err, context.DeadlineExceeded) {
			return fmt.Sprintf("timeout after %s", timeout)
		}
		if ue.Err != nil {
			err = ue.Err
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Sprintf("timeout after %s", timeout)
	}
	return err.Error()
}
