package uptime

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/model"
)

func sampleEvent() Event {
	return Event{
		MonitorID:  uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Name:       "API",
		URL:        "https://api.example.com/health",
		Status:     model.MonitorDown,
		Message:    "unexpected status 503 Service Unavailable",
		HTTPStatus: 503,
		LatencyMS:  120,
		CheckedAt:  time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC),
		Kind:       KindStatus,
	}
}

type captured struct {
	method string
	header http.Header
	body   string
}

func captureServer(t *testing.T) (*httptest.Server, *captured) {
	t.Helper()
	c := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		c.method, c.header, c.body = r.Method, r.Header.Clone(), string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return srv, c
}

func TestDiscordTextPayload(t *testing.T) {
	cfg := &model.DiscordConfig{BotDisplayName: "Nocu", PrefixCustomMessage: "@here", MessageFormat: model.DiscordFormatText}
	p := discordPayload(cfg, sampleEvent())
	if p["username"] != "Nocu" {
		t.Fatalf("username: %v", p["username"])
	}
	content := p["content"].(string)
	if !strings.HasPrefix(content, "@here\nAPI is DOWN (https://api.example.com/health)") || !strings.Contains(content, "503") {
		t.Fatalf("content: %q", content)
	}
	if _, ok := p["embeds"]; ok {
		t.Fatal("text format must not include embeds")
	}

	cfg.DisableURL = true
	content = discordPayload(cfg, sampleEvent())["content"].(string)
	if strings.Contains(content, "https://") {
		t.Fatalf("url should be hidden: %q", content)
	}
}

func TestDiscordEmbedPayload(t *testing.T) {
	cfg := &model.DiscordConfig{MessageFormat: model.DiscordFormatEmbed, DisableURL: true, PrefixCustomMessage: "prod"}
	p := discordPayload(cfg, sampleEvent())
	if _, ok := p["content"]; ok {
		t.Fatal("embed format must not include content")
	}
	if _, ok := p["username"]; ok {
		t.Fatal("username should be absent when bot name is empty")
	}
	embeds := p["embeds"].([]any)
	e := embeds[0].(map[string]any)
	if e["title"] != "API is DOWN" || e["color"] != colorDown || e["description"] != "prod" {
		t.Fatalf("embed: %v", e)
	}
	for _, f := range e["fields"].([]map[string]any) {
		if f["name"] == "URL" {
			t.Fatal("URL field should be hidden")
		}
	}
	up := sampleEvent()
	up.Status = model.MonitorUp
	if discordPayload(cfg, up)["embeds"].([]any)[0].(map[string]any)["color"] != colorUp {
		t.Fatal("up colour")
	}
	exp := sampleEvent()
	exp.Kind = KindCertExpiring
	exp.DaysLeft = 7
	e = discordPayload(cfg, exp)["embeds"].([]any)[0].(map[string]any)
	if e["color"] != colorWarning || e["title"] != "API: TLS certificate expires in 7 days" {
		t.Fatalf("expiry embed: %v", e)
	}
}

func TestDiscordSenderPostsJSON(t *testing.T) {
	srv, c := captureServer(t)
	cfg := &model.DiscordConfig{WebhookURL: srv.URL + "/api/webhooks/1/x", MessageFormat: model.DiscordFormatText}
	s := &DiscordSender{Client: srv.Client()}
	if err := s.Send(context.Background(), cfg, sampleEvent()); err != nil {
		t.Fatal(err)
	}
	if c.method != http.MethodPost || c.header.Get("Content-Type") != "application/json" {
		t.Fatalf("request: %s %v", c.method, c.header)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(c.body), &body); err != nil || body["content"] == "" {
		t.Fatalf("body: %s %v", c.body, err)
	}
}

func TestWebhookJSON(t *testing.T) {
	srv, c := captureServer(t)
	cfg := &model.WebhookConfig{PostURL: srv.URL, HTTPMethod: model.MethodPOST, RequestBody: model.WebhookBodyJSON,
		AdditionalHeaders: map[string]string{"X-Token": "abc"}}
	if err := (&WebhookSender{Client: srv.Client()}).Send(context.Background(), cfg, sampleEvent()); err != nil {
		t.Fatal(err)
	}
	if c.method != http.MethodPost || c.header.Get("X-Token") != "abc" || c.header.Get("Content-Type") != "application/json" {
		t.Fatalf("request: %s %v", c.method, c.header)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(c.body), &body); err != nil {
		t.Fatal(err)
	}
	if body["monitor_name"] != "API" || body["status"] != "down" || body["http_status"] != float64(503) || body["kind"] != "status" {
		t.Fatalf("payload: %v", body)
	}
	if _, ok := body["expires_at"]; ok {
		t.Fatal("expires_at must be absent for status events")
	}
}

func TestWebhookForm(t *testing.T) {
	srv, c := captureServer(t)
	cfg := &model.WebhookConfig{PostURL: srv.URL, HTTPMethod: model.MethodPUT, RequestBody: model.WebhookBodyForm}
	if err := (&WebhookSender{Client: srv.Client()}).Send(context.Background(), cfg, sampleEvent()); err != nil {
		t.Fatal(err)
	}
	if c.method != http.MethodPut || c.header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("request: %s %v", c.method, c.header)
	}
	v, err := url.ParseQuery(c.body)
	if err != nil {
		t.Fatal(err)
	}
	if v.Get("monitor_name") != "API" || v.Get("status") != "down" || v.Get("latency_ms") != "120" || v.Get("monitor_id") != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("form: %v", v)
	}
}

func TestWebhookCustomBody(t *testing.T) {
	srv, c := captureServer(t)
	cfg := &model.WebhookConfig{PostURL: srv.URL, HTTPMethod: model.MethodPATCH, RequestBody: model.WebhookBodyCustom,
		CustomBody:        `text={{monitor_name}} {{status}} {{monitor_url}} :: {{message}}`,
		AdditionalHeaders: map[string]string{"Content-Type": "text/plain"}}
	if err := (&WebhookSender{Client: srv.Client()}).Send(context.Background(), cfg, sampleEvent()); err != nil {
		t.Fatal(err)
	}
	if c.method != http.MethodPatch || c.header.Get("Content-Type") != "text/plain" {
		t.Fatalf("request: %s %v", c.method, c.header)
	}
	want := "text=API down https://api.example.com/health :: unexpected status 503 Service Unavailable"
	if c.body != want {
		t.Fatalf("body %q want %q", c.body, want)
	}
}

func TestWebhookNon2xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer srv.Close()
	cfg := &model.WebhookConfig{PostURL: srv.URL, HTTPMethod: model.MethodPOST, RequestBody: model.WebhookBodyJSON}
	err := (&WebhookSender{Client: srv.Client()}).Send(context.Background(), cfg, sampleEvent())
	if err == nil || !strings.Contains(err.Error(), "502") {
		t.Fatalf("expected 502 error, got %v", err)
	}
}

func TestSendersRejectWrongConfigType(t *testing.T) {
	if err := (&DiscordSender{}).Send(context.Background(), &model.WebhookConfig{}, sampleEvent()); err == nil {
		t.Fatal("discord should reject webhook config")
	}
	if err := (&WebhookSender{}).Send(context.Background(), &model.DiscordConfig{}, sampleEvent()); err == nil {
		t.Fatal("webhook should reject discord config")
	}
	if err := (WebPushSender{}).Send(context.Background(), &model.WebPushConfig{}, sampleEvent()); err != ErrNotSupported {
		t.Fatalf("webpush: %v", err)
	}
}
