package monitor

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

type fakeRepo struct {
	items      map[uuid.UUID]*model.Monitor
	heartbeats map[uuid.UUID][]model.MonitorHeartbeat
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{items: map[uuid.UUID]*model.Monitor{}, heartbeats: map[uuid.UUID][]model.MonitorHeartbeat{}}
}

func clone(m *model.Monitor) *model.Monitor {
	cp := *m
	cp.Notifications = append([]model.MonitorNotification(nil), m.Notifications...)
	return &cp
}

func (f *fakeRepo) Create(_ context.Context, m *model.Monitor) error {
	m.ID = uuid.New()
	for i := range m.Notifications {
		m.Notifications[i].ID = uuid.New()
		m.Notifications[i].MonitorID = m.ID
	}
	f.items[m.ID] = clone(m)
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (*model.Monitor, error) {
	if m, ok := f.items[id]; ok {
		return clone(m), nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, _ ListFilter) ([]model.Monitor, int64, error) {
	out := make([]model.Monitor, 0, len(f.items))
	for _, m := range f.items {
		out = append(out, *clone(m))
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) Update(_ context.Context, m *model.Monitor) error {
	if _, ok := f.items[m.ID]; !ok {
		return ErrNotFound
	}
	for i := range m.Notifications {
		m.Notifications[i].ID = uuid.New()
		m.Notifications[i].MonitorID = m.ID
	}
	f.items[m.ID] = clone(m)
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.items[id]; !ok {
		return ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func (f *fakeRepo) ListHeartbeats(_ context.Context, id uuid.UUID, _ HeartbeatFilter) ([]model.MonitorHeartbeat, int64, error) {
	hb := f.heartbeats[id]
	return hb, int64(len(hb)), nil
}

func newService() (*Service, *fakeRepo, *messaging.Memory) {
	broker := messaging.NewMemory()
	repo := newFakeRepo()
	svc := NewService(repo, broker, slog.Default())
	svc.now = func() time.Time { return time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC) }
	return svc, repo, broker
}

func ptr[T any](v T) *T { return &v }

func validCreate() CreateInput {
	return CreateInput{Name: "  API  ", URL: "https://example.com/health"}
}

func TestCreateAppliesDefaults(t *testing.T) {
	svc, _, broker := newService()
	m, err := svc.Create(context.Background(), validCreate())
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "API" || m.Type != model.MonitorHTTP || !m.Active || m.LastStatus != model.MonitorPending {
		t.Fatalf("unexpected monitor: %+v", m)
	}
	if m.IntervalSeconds != model.MonitorDefaultInterval || m.TimeoutSeconds != model.MonitorDefaultTimeout ||
		m.Retries != 0 || m.MaxRedirects != model.MonitorDefaultMaxRedirects {
		t.Fatalf("defaults not applied: %+v", m)
	}
	if m.Method != model.MethodGET || m.BodyEncoding != model.BodyJSON || m.AuthMethod != model.AuthNone {
		t.Fatalf("http defaults not applied: %+v", m)
	}
	if string(m.Headers) != "{}" {
		t.Fatalf("headers should default to {}, got %s", m.Headers)
	}
	if m.NextCheckAt == nil || !m.NextCheckAt.Equal(svc.now()) {
		t.Fatalf("next_check_at should be now: %v", m.NextCheckAt)
	}
	if m.Notifications == nil || len(m.Notifications) != 0 {
		t.Fatalf("notifications should be an empty slice: %v", m.Notifications)
	}
	if subs := broker.Subjects(); len(subs) != 1 || subs[0] != messaging.SubjectMonitorCreated {
		t.Fatalf("expected monitor.created, got %v", subs)
	}
}

func TestCreateValidation(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	cases := []struct {
		name string
		mod  func(in *CreateInput)
		want string
	}{
		{"bad type", func(in *CreateInput) { in.Type = "ping" }, "type"},
		{"empty name", func(in *CreateInput) { in.Name = "  " }, "name"},
		{"empty url", func(in *CreateInput) { in.URL = "" }, "url"},
		{"ftp url", func(in *CreateInput) { in.URL = "ftp://example.com" }, "url"},
		{"no host", func(in *CreateInput) { in.URL = "https://" }, "url"},
		{"interval too small", func(in *CreateInput) { in.IntervalSeconds = ptr(5); in.TimeoutSeconds = ptr(1) }, "interval"},
		{"negative retries", func(in *CreateInput) { in.Retries = ptr(-1) }, "retries"},
		{"zero timeout", func(in *CreateInput) { in.TimeoutSeconds = ptr(0) }, "timeout"},
		{"timeout >= interval", func(in *CreateInput) { in.IntervalSeconds = ptr(30); in.TimeoutSeconds = ptr(30) }, "timeout"},
		{"negative redirects", func(in *CreateInput) { in.MaxRedirects = ptr(-1) }, "max_redirects"},
		{"bad method", func(in *CreateInput) { in.Method = "FETCH" }, "method"},
		{"bad encoding", func(in *CreateInput) { in.BodyEncoding = "yaml" }, "body_encoding"},
		{"headers not object", func(in *CreateInput) { in.Headers = json.RawMessage(`["a"]`) }, "headers"},
		{"headers non-string", func(in *CreateInput) { in.Headers = json.RawMessage(`{"a":1}`) }, "headers"},
		{"bad auth", func(in *CreateInput) { in.AuthMethod = "digest" }, "auth_method"},
		{"basic without user", func(in *CreateInput) { in.AuthMethod = model.AuthBasic }, "basic_username"},
		{"bearer without token", func(in *CreateInput) { in.AuthMethod = model.AuthBearer }, "bearer_token"},
		{"bad notification type", func(in *CreateInput) {
			in.Notifications = []NotificationInput{{Type: "sms", Config: json.RawMessage(`{}`)}}
		}, "notifications[0]"},
		{"bad notification config", func(in *CreateInput) {
			in.Notifications = []NotificationInput{{Type: model.NotificationDiscord, Config: json.RawMessage(`{"webhook_url":"http://x"}`)}}
		}, "notifications[0]"},
		{"unknown notification field", func(in *CreateInput) {
			in.Notifications = []NotificationInput{{Type: model.NotificationWebhook, Config: json.RawMessage(`{"post_url":"https://x","oops":1}`)}}
		}, "notifications[0]"},
	}
	for _, c := range cases {
		in := validCreate()
		c.mod(&in)
		_, err := svc.Create(ctx, in)
		if !errors.Is(err, ErrValidation) {
			t.Errorf("%s: expected ErrValidation, got %v", c.name, err)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: error %q should mention %q", c.name, err, c.want)
		}
	}
}

func TestCreateStoresSecretsButHidesThemInJSON(t *testing.T) {
	svc, repo, _ := newService()
	in := validCreate()
	in.AuthMethod = model.AuthBasic
	in.BasicUsername = "ops"
	in.BasicPassword = "s3cret"
	m, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	stored := repo.items[m.ID]
	if stored.BasicPassword != "s3cret" || stored.BasicUsername != "ops" {
		t.Fatalf("secret not stored: %+v", stored)
	}
	out, _ := json.Marshal(m)
	if strings.Contains(string(out), "s3cret") {
		t.Fatalf("secret leaked: %s", out)
	}
	if !strings.Contains(string(out), `"has_basic_password":true`) {
		t.Fatalf("has_basic_password missing: %s", out)
	}
}

func TestCreateIgnoresSecretsWhenAuthNone(t *testing.T) {
	svc, _, _ := newService()
	in := validCreate()
	in.BasicUsername, in.BasicPassword, in.BearerToken = "u", "p", "t"
	m, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if m.BasicUsername != "" || m.BasicPassword != "" || m.BearerToken != "" {
		t.Fatalf("secrets should be cleared for auth none: %+v", m)
	}
}

func TestNotificationDefaultsApplied(t *testing.T) {
	svc, _, _ := newService()
	in := validCreate()
	in.Notifications = []NotificationInput{
		{Type: model.NotificationWebhook, Config: json.RawMessage(`{"post_url":"https://hooks.example/x"}`)},
		{Type: model.NotificationDiscord, Enabled: ptr(false), Config: json.RawMessage(`{"webhook_url":"https://discord.com/api/webhooks/1/a"}`)},
	}
	m, err := svc.Create(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Notifications) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(m.Notifications))
	}
	var wh model.WebhookConfig
	if err := json.Unmarshal(m.Notifications[0].Config, &wh); err != nil {
		t.Fatal(err)
	}
	if wh.HTTPMethod != model.MethodPOST || wh.RequestBody != model.WebhookBodyJSON || !m.Notifications[0].Enabled {
		t.Fatalf("webhook defaults not applied: %+v enabled=%v", wh, m.Notifications[0].Enabled)
	}
	var dc model.DiscordConfig
	if err := json.Unmarshal(m.Notifications[1].Config, &dc); err != nil {
		t.Fatal(err)
	}
	if dc.MessageFormat != model.DiscordFormatText || m.Notifications[1].Enabled {
		t.Fatalf("discord defaults not applied: %+v enabled=%v", dc, m.Notifications[1].Enabled)
	}
}

func TestUpdateReplacesNotificationsAndReschedules(t *testing.T) {
	svc, repo, broker := newService()
	ctx := context.Background()
	in := validCreate()
	in.Notifications = []NotificationInput{
		{Type: model.NotificationWebhook, Config: json.RawMessage(`{"post_url":"https://hooks.example/x"}`)},
	}
	m, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	// Simulate the scheduler having probed it.
	later := svc.now().Add(time.Minute)
	repo.items[m.ID].NextCheckAt = &later
	repo.items[m.ID].FailureCount = 2

	// Non-probe change keeps schedule.
	upd, err := svc.Update(ctx, m.ID, UpdateInput{Name: ptr("renamed")})
	if err != nil {
		t.Fatal(err)
	}
	if upd.Name != "renamed" || !upd.NextCheckAt.Equal(later) || upd.FailureCount != 2 {
		t.Fatalf("non-probe change should keep schedule: %+v", upd)
	}
	if len(upd.Notifications) != 1 {
		t.Fatalf("notifications should be untouched when nil: %v", upd.Notifications)
	}

	// Probe-affecting change with notification replacement.
	upd, err = svc.Update(ctx, m.ID, UpdateInput{
		URL: ptr("https://example.com/v2"),
		Notifications: &[]NotificationInput{
			{Type: model.NotificationWebPush, Config: json.RawMessage(`{"endpoint":"https://push.example/1","keys":{"p256dh":"p","auth":"a"}}`)},
			{Type: model.NotificationDiscord, Config: json.RawMessage(`{"webhook_url":"https://discord.com/api/webhooks/1/a","message_format":"embed"}`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !upd.NextCheckAt.Equal(svc.now()) || upd.FailureCount != 0 {
		t.Fatalf("probe change should reschedule now: %+v", upd)
	}
	if len(upd.Notifications) != 2 || upd.Notifications[0].Type != model.NotificationWebPush || upd.Notifications[1].Type != model.NotificationDiscord {
		t.Fatalf("notifications not replaced: %+v", upd.Notifications)
	}

	// Clear all notifications with an empty list.
	upd, err = svc.Update(ctx, m.ID, UpdateInput{Notifications: &[]NotificationInput{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(upd.Notifications) != 0 {
		t.Fatalf("notifications should be cleared: %+v", upd.Notifications)
	}

	want := []string{messaging.SubjectMonitorCreated, messaging.SubjectMonitorUpdated, messaging.SubjectMonitorUpdated, messaging.SubjectMonitorUpdated}
	got := broker.Subjects()
	if len(got) != len(want) {
		t.Fatalf("published %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published %v want %v", got, want)
		}
	}
}

func TestUpdateValidation(t *testing.T) {
	svc, _, _ := newService()
	ctx := context.Background()
	m, _ := svc.Create(ctx, validCreate())
	if _, err := svc.Update(ctx, m.ID, UpdateInput{Name: ptr(" ")}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty name: %v", err)
	}
	if _, err := svc.Update(ctx, m.ID, UpdateInput{TimeoutSeconds: ptr(120)}); !errors.Is(err, ErrValidation) {
		t.Fatalf("timeout >= interval: %v", err)
	}
	if _, err := svc.Update(ctx, m.ID, UpdateInput{Headers: json.RawMessage(`"x"`)}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad headers: %v", err)
	}
	if _, err := svc.Update(ctx, uuid.New(), UpdateInput{Name: ptr("x")}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing monitor: %v", err)
	}
}

func TestUpdateAuthSwitchingAndClearing(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	in := validCreate()
	in.AuthMethod = model.AuthBearer
	in.BearerToken = "tok"
	m, err := svc.Create(ctx, in)
	if err != nil {
		t.Fatal(err)
	}

	// Switch to basic: bearer dropped, username required.
	if _, err := svc.Update(ctx, m.ID, UpdateInput{AuthMethod: ptr(model.AuthBasic)}); !errors.Is(err, ErrValidation) {
		t.Fatalf("basic without username should fail: %v", err)
	}
	upd, err := svc.Update(ctx, m.ID, UpdateInput{AuthMethod: ptr(model.AuthBasic), BasicUsername: ptr("ops"), BasicPassword: ptr("pw")})
	if err != nil {
		t.Fatal(err)
	}
	if upd.BearerToken != "" || upd.BasicUsername != "ops" || upd.BasicPassword != "pw" {
		t.Fatalf("switch to basic wrong: %+v", upd)
	}

	// nil password leaves it, "" clears it.
	upd, err = svc.Update(ctx, m.ID, UpdateInput{BasicUsername: ptr("ops2")})
	if err != nil {
		t.Fatal(err)
	}
	if upd.BasicPassword != "pw" {
		t.Fatalf("nil password should be unchanged: %+v", upd)
	}
	upd, err = svc.Update(ctx, m.ID, UpdateInput{BasicPassword: ptr("")})
	if err != nil {
		t.Fatal(err)
	}
	if upd.BasicPassword != "" || repo.items[m.ID].BasicPassword != "" {
		t.Fatalf("empty password should clear: %+v", upd)
	}

	// Switch to none clears everything.
	upd, err = svc.Update(ctx, m.ID, UpdateInput{AuthMethod: ptr(model.AuthNone)})
	if err != nil {
		t.Fatal(err)
	}
	if upd.BasicUsername != "" || upd.BasicPassword != "" || upd.BearerToken != "" || upd.AuthMethod != model.AuthNone {
		t.Fatalf("switch to none should clear: %+v", upd)
	}
}

func TestPauseResume(t *testing.T) {
	svc, repo, broker := newService()
	ctx := context.Background()
	m, _ := svc.Create(ctx, validCreate())
	repo.items[m.ID].FailureCount = 3

	paused, err := svc.Pause(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if paused.Active || paused.NextCheckAt != nil {
		t.Fatalf("pause not applied: %+v", paused)
	}
	// Second pause is a no-op and publishes nothing.
	before := len(broker.Subjects())
	if _, err := svc.Pause(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if len(broker.Subjects()) != before {
		t.Fatal("no-op pause should not publish")
	}

	resumed, err := svc.Resume(ctx, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !resumed.Active || resumed.NextCheckAt == nil || !resumed.NextCheckAt.Equal(svc.now()) || resumed.FailureCount != 0 {
		t.Fatalf("resume not applied: %+v", resumed)
	}
	if _, err := svc.Resume(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("resume missing: %v", err)
	}
}

func TestListValidationAndHeartbeats(t *testing.T) {
	svc, repo, _ := newService()
	ctx := context.Background()
	if _, _, err := svc.List(ctx, ListFilter{Status: "flapping"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad status: %v", err)
	}
	m, _ := svc.Create(ctx, validCreate())
	repo.heartbeats[m.ID] = []model.MonitorHeartbeat{{MonitorID: m.ID, Status: model.MonitorUp}}
	hb, total, err := svc.ListHeartbeats(ctx, m.ID, HeartbeatFilter{})
	if err != nil || total != 1 || len(hb) != 1 {
		t.Fatalf("heartbeats: %v %d %v", hb, total, err)
	}
	if _, _, err := svc.ListHeartbeats(ctx, uuid.New(), HeartbeatFilter{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("heartbeats of missing monitor: %v", err)
	}
}

func TestDelete(t *testing.T) {
	svc, _, broker := newService()
	ctx := context.Background()
	if err := svc.Delete(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	m, _ := svc.Create(ctx, validCreate())
	if err := svc.Delete(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("should be gone: %v", err)
	}
	subs := broker.Subjects()
	if subs[len(subs)-1] != messaging.SubjectMonitorDeleted {
		t.Fatalf("expected monitor.deleted last, got %v", subs)
	}
}
