package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMonitorEnumsValid(t *testing.T) {
	if !MonitorHTTP.Valid() || MonitorType("ping").Valid() {
		t.Fatal("monitor type validation broken")
	}
	if !MethodGET.Valid() || HTTPMethod("FETCH").Valid() {
		t.Fatal("http method validation broken")
	}
	if !BodyForm.Valid() || BodyEncoding("yaml").Valid() {
		t.Fatal("body encoding validation broken")
	}
	if !AuthBearer.Valid() || AuthMethod("digest").Valid() {
		t.Fatal("auth method validation broken")
	}
	if !MonitorDown.Valid() || MonitorStatus("flapping").Valid() {
		t.Fatal("monitor status validation broken")
	}
	if !NotificationWebhook.Valid() || NotificationType("sms").Valid() {
		t.Fatal("notification type validation broken")
	}
}

func TestMonitorJSONHidesSecrets(t *testing.T) {
	m := Monitor{Name: "api", BasicPassword: "s3cret", BearerToken: "tok"}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "s3cret") || strings.Contains(s, "tok\"") {
		t.Fatalf("secrets leaked: %s", s)
	}
	if !strings.Contains(s, `"has_basic_password":true`) || !strings.Contains(s, `"has_bearer_token":true`) {
		t.Fatalf("missing has_* flags: %s", s)
	}
}

func TestDiscordConfigValidate(t *testing.T) {
	cases := []struct {
		name string
		cfg  DiscordConfig
		ok   bool
	}{
		{"valid", DiscordConfig{WebhookURL: "https://discord.com/api/webhooks/1/abc"}, true},
		{"discordapp", DiscordConfig{WebhookURL: "https://discordapp.com/api/webhooks/1/abc", MessageFormat: DiscordFormatEmbed}, true},
		{"http", DiscordConfig{WebhookURL: "http://discord.com/api/webhooks/1/abc"}, false},
		{"wrong host", DiscordConfig{WebhookURL: "https://example.com/api/webhooks/1/abc"}, false},
		{"wrong path", DiscordConfig{WebhookURL: "https://discord.com/other"}, false},
		{"bad format", DiscordConfig{WebhookURL: "https://discord.com/api/webhooks/1/abc", MessageFormat: "html"}, false},
		{"empty", DiscordConfig{}, false},
	}
	for _, c := range cases {
		err := c.cfg.Validate()
		if (err == nil) != c.ok {
			t.Errorf("%s: got %v want ok=%v", c.name, err, c.ok)
		}
		if c.ok && c.cfg.MessageFormat == "" {
			t.Errorf("%s: default message_format not applied", c.name)
		}
	}
}

func TestWebPushConfigValidate(t *testing.T) {
	good := WebPushConfig{Endpoint: "https://push.example/sub/1", Keys: WebPushKeys{P256dh: "p", Auth: "a"}}
	if err := good.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := []WebPushConfig{
		{Endpoint: "http://push.example/sub/1", Keys: WebPushKeys{P256dh: "p", Auth: "a"}},
		{Endpoint: "https://push.example/sub/1", Keys: WebPushKeys{P256dh: "", Auth: "a"}},
		{Endpoint: "https://push.example/sub/1"},
		{},
	}
	for i, b := range bad {
		if err := b.Validate(); err == nil {
			t.Errorf("case %d should fail", i)
		}
	}
}

func TestWebhookConfigValidate(t *testing.T) {
	c := WebhookConfig{PostURL: "https://hooks.example/x"}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	if c.HTTPMethod != MethodPOST || c.RequestBody != WebhookBodyJSON {
		t.Fatalf("defaults not applied: %+v", c)
	}
	bad := []WebhookConfig{
		{PostURL: "ftp://hooks.example/x"},
		{PostURL: "https://hooks.example/x", HTTPMethod: MethodGET},
		{PostURL: "https://hooks.example/x", RequestBody: "yaml"},
		{PostURL: "https://hooks.example/x", RequestBody: WebhookBodyCustom},
		{PostURL: "https://hooks.example/x", AdditionalHeaders: map[string]string{" ": "v"}},
	}
	for i, b := range bad {
		if err := b.Validate(); err == nil {
			t.Errorf("case %d should fail", i)
		}
	}
	custom := WebhookConfig{PostURL: "https://hooks.example/x", RequestBody: WebhookBodyCustom, CustomBody: `{"m":"{{monitor_name}}"}`}
	if err := custom.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestDecodeNotificationConfig(t *testing.T) {
	cfg, err := DecodeNotificationConfig(NotificationDiscord, []byte(`{"webhook_url":"https://discord.com/api/webhooks/1/a"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.(*DiscordConfig); !ok {
		t.Fatalf("wrong type %T", cfg)
	}
	if _, err := DecodeNotificationConfig(NotificationWebhook, []byte(`{"post_url":"https://x","typo":1}`)); err == nil {
		t.Fatal("unknown field should be rejected")
	}
	if _, err := DecodeNotificationConfig("sms", []byte(`{}`)); err == nil {
		t.Fatal("unknown type should be rejected")
	}
	if _, err := DecodeNotificationConfig(NotificationWebPush, nil); err == nil {
		t.Fatal("empty config should be rejected")
	}
}

func TestMonitorHeaderMap(t *testing.T) {
	m := Monitor{}
	h, err := m.HeaderMap()
	if err != nil || len(h) != 0 {
		t.Fatalf("empty headers: %v %v", h, err)
	}
	m.Headers = []byte(`{"X-A":"1"}`)
	h, err = m.HeaderMap()
	if err != nil || h["X-A"] != "1" {
		t.Fatalf("headers: %v %v", h, err)
	}
	m.Headers = []byte(`["nope"]`)
	if _, err := m.HeaderMap(); err == nil {
		t.Fatal("non-object headers should fail")
	}
}
