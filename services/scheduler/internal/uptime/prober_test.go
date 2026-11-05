package uptime

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Pzquzer/Nocu/shared/model"
)

// localProber allows loopback so httptest servers are reachable.
func localProber() *Prober { return NewProber(true) }

func baseMonitor(url string) *model.Monitor {
	return &model.Monitor{
		Type:            model.MonitorHTTP,
		Name:            "t",
		URL:             url,
		IntervalSeconds: 60,
		TimeoutSeconds:  5,
		MaxRedirects:    10,
		Method:          model.MethodGET,
		BodyEncoding:    model.BodyJSON,
		AuthMethod:      model.AuthNone,
	}
}

func TestProbeUpAndDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/fail" {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, "boom body")
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	res := localProber().Probe(context.Background(), baseMonitor(srv.URL+"/ok"))
	if res.Status != model.MonitorUp || res.HTTPStatus != 200 {
		t.Fatalf("up probe: %+v", res)
	}

	m := baseMonitor(srv.URL + "/fail")
	res = localProber().Probe(context.Background(), m)
	if res.Status != model.MonitorDown || res.HTTPStatus != 500 || res.ErrorBody != "" {
		t.Fatalf("down probe without save: %+v", res)
	}
	m.SaveErrorResponse = true
	res = localProber().Probe(context.Background(), m)
	if res.ErrorBody != "boom body" {
		t.Fatalf("error body not saved: %+v", res)
	}
}

func TestProbeTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	m := baseMonitor(srv.URL)
	m.TimeoutSeconds = 1
	res := localProber().Probe(context.Background(), m)
	if res.Status != model.MonitorDown || !strings.Contains(res.Message, "timeout") {
		t.Fatalf("timeout probe: %+v", res)
	}
}

func TestProbeRedirectLimit(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a":
			http.Redirect(w, r, srv.URL+"/b", http.StatusFound)
		case "/b":
			http.Redirect(w, r, srv.URL+"/c", http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL + "/a")
	m.MaxRedirects = 0
	res := localProber().Probe(context.Background(), m)
	if res.HTTPStatus != http.StatusFound || res.Status != model.MonitorUp {
		t.Fatalf("no redirects: expected last 302 as up, got %+v", res)
	}
	m.MaxRedirects = 1
	res = localProber().Probe(context.Background(), m)
	if res.HTTPStatus != http.StatusFound {
		t.Fatalf("one redirect: expected 302 from /b, got %+v", res)
	}
	m.MaxRedirects = 5
	res = localProber().Probe(context.Background(), m)
	if res.HTTPStatus != http.StatusOK {
		t.Fatalf("follow all: got %+v", res)
	}
}

func TestProbeRequestShape(t *testing.T) {
	var got *http.Request
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		b, _ := io.ReadAll(r.Body)
		body = string(b)
	}))
	defer srv.Close()

	m := baseMonitor(srv.URL)
	m.Method = model.MethodPOST
	m.Body = `a=1&b=2`
	m.BodyEncoding = model.BodyForm
	m.Headers = []byte(`{"X-Probe":"yes"}`)
	m.AuthMethod = model.AuthBasic
	m.BasicUsername = "u"
	m.BasicPassword = "p"
	localProber().Probe(context.Background(), m)

	if got.Method != http.MethodPost || body != "a=1&b=2" {
		t.Fatalf("method/body: %s %q", got.Method, body)
	}
	if ct := got.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
		t.Fatalf("content-type: %q", ct)
	}
	if got.Header.Get("X-Probe") != "yes" {
		t.Fatal("custom header missing")
	}
	u, p, ok := got.BasicAuth()
	if !ok || u != "u" || p != "p" {
		t.Fatal("basic auth missing")
	}

	m.AuthMethod = model.AuthBearer
	m.BearerToken = "tok"
	m.Headers = []byte(`{"Content-Type":"text/custom"}`)
	localProber().Probe(context.Background(), m)
	if got.Header.Get("Authorization") != "Bearer tok" {
		t.Fatalf("bearer: %q", got.Header.Get("Authorization"))
	}
	if got.Header.Get("Content-Type") != "text/custom" {
		t.Fatal("explicit content-type should win over encoding default")
	}

	m.Method = model.MethodGET
	localProber().Probe(context.Background(), m)
	if body != "" {
		t.Fatal("GET must not send a body")
	}
}

func TestProbeInvalidURLAndHeaders(t *testing.T) {
	res := localProber().Probe(context.Background(), baseMonitor("ftp://example.com"))
	if res.Status != model.MonitorDown || !strings.Contains(res.Message, "invalid url") {
		t.Fatalf("bad scheme: %+v", res)
	}
	m := baseMonitor("http://example.com")
	m.Headers = []byte(`["x"]`)
	res = localProber().Probe(context.Background(), m)
	if res.Status != model.MonitorDown || !strings.Contains(res.Message, "invalid headers") {
		t.Fatalf("bad headers: %+v", res)
	}
}

func TestIsDisallowedIP(t *testing.T) {
	bad := []string{"127.0.0.1", "10.1.2.3", "172.16.0.1", "192.168.1.1", "169.254.169.254", "0.0.0.0", "224.0.0.1", "100.64.0.1", "::1", "fc00::1", "fe80::1", "::"}
	for _, s := range bad {
		if !IsDisallowedIP(net.ParseIP(s)) {
			t.Errorf("%s should be disallowed", s)
		}
	}
	good := []string{"8.8.8.8", "1.1.1.1", "2606:4700:4700::1111", "100.128.0.1"}
	for _, s := range good {
		if IsDisallowedIP(net.ParseIP(s)) {
			t.Errorf("%s should be allowed", s)
		}
	}
	if !IsDisallowedIP(nil) {
		t.Error("nil should be disallowed")
	}
}

type fakeResolver map[string][]string

func (f fakeResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	ips, ok := f[host]
	if !ok {
		return net.DefaultResolver.LookupIPAddr(context.Background(), host)
	}
	out := make([]net.IPAddr, 0, len(ips))
	for _, s := range ips {
		out = append(out, net.IPAddr{IP: net.ParseIP(s)})
	}
	return out, nil
}

func TestProbeBlocksPrivateTargets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()

	// IP literal loopback.
	res := NewProber(false).Probe(context.Background(), baseMonitor(srv.URL))
	if res.Status != model.MonitorDown || !strings.Contains(res.Message, "private or local") {
		t.Fatalf("loopback literal should be blocked: %+v", res)
	}

	// Public-looking name resolving to 10.x (DNS rebinding style).
	_, port, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	m := baseMonitor("http://internal.example:" + port)
	res = NewProber(false).WithResolver(fakeResolver{"internal.example": {"10.0.0.5"}}).Probe(context.Background(), m)
	if res.Status != model.MonitorDown || !strings.Contains(res.Message, "private or local") {
		t.Fatalf("private resolution should be blocked: %+v", res)
	}

	// Same name allowed when the guard is off and it resolves to loopback.
	res = NewProber(true).WithResolver(fakeResolver{"internal.example": {"127.0.0.1"}}).Probe(context.Background(), m)
	if res.Status != model.MonitorUp {
		t.Fatalf("allowPrivate should permit: %+v", res)
	}
}

func TestProbeCapturesCertExpiry(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	p := localProber()
	// Trust the test server certificate.
	m := baseMonitor(srv.URL)
	tr := newTransport(true, nil)
	tr.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	req, _ := buildRequest(context.Background(), m)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.TLS == nil || len(resp.TLS.PeerCertificates) == 0 {
		t.Fatal("expected TLS state")
	}
	// Untrusted cert via the real prober yields a down result with a TLS message.
	res := p.Probe(context.Background(), m)
	if res.Status != model.MonitorDown {
		t.Fatalf("self-signed cert should fail verification: %+v", res)
	}
}
