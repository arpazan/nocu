package uptime

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

type fakeStore struct {
	mu         sync.Mutex
	monitors   []model.Monitor
	heartbeats []model.MonitorHeartbeat
	updates    map[uuid.UUID][]map[string]any
}

func newFakeStore(ms ...model.Monitor) *fakeStore {
	return &fakeStore{monitors: ms, updates: map[uuid.UUID][]map[string]any{}}
}

func (f *fakeStore) Due(_ context.Context, now time.Time, limit int) ([]model.Monitor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Monitor
	for i := range f.monitors {
		m := &f.monitors[i]
		if !m.Active || (m.NextCheckAt != nil && m.NextCheckAt.After(now)) {
			continue
		}
		next := now.Add(m.Interval())
		m.NextCheckAt = &next
		out = append(out, *m)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (f *fakeStore) ExpiryCandidates(context.Context) ([]model.Monitor, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []model.Monitor
	for _, m := range f.monitors {
		if m.Active && (m.CertExpiryNotification || m.DomainExpiryNotification) {
			out = append(out, m)
		}
	}
	return out, nil
}

func (f *fakeStore) SaveHeartbeat(_ context.Context, hb *model.MonitorHeartbeat) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	hb.ID = uuid.New()
	f.heartbeats = append(f.heartbeats, *hb)
	return nil
}

func (f *fakeStore) UpdateMonitor(_ context.Context, id uuid.UUID, fields map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updates[id] = append(f.updates[id], fields)
	for i := range f.monitors {
		m := &f.monitors[i]
		if m.ID != id {
			continue
		}
		if v, ok := fields["last_status"]; ok {
			m.LastStatus = v.(model.MonitorStatus)
		}
		if v, ok := fields["failure_count"]; ok {
			m.FailureCount = v.(int)
		}
		if v, ok := fields["domain_expires_at"]; ok {
			t := v.(time.Time)
			m.DomainExpiresAt = &t
		}
	}
	return nil
}

func TestTransitionRetriesThenDown(t *testing.T) {
	m := &model.Monitor{Base: model.Base{ID: uuid.New()}, Retries: 2, LastStatus: model.MonitorUp}
	at := time.Now()
	down := Result{Status: model.MonitorDown, Message: "500"}

	hb, fields := transition(m, down, at)
	if hb.Status != model.MonitorPending || hb.Important || hb.Message != "retry 1/2: 500" {
		t.Fatalf("first failure: %+v", hb)
	}
	if fields["last_status"] != model.MonitorUp || fields["failure_count"] != 1 {
		t.Fatalf("fields after first failure: %v", fields)
	}
	hb, _ = transition(m, down, at)
	if hb.Status != model.MonitorPending || hb.Important {
		t.Fatalf("second failure: %+v", hb)
	}
	hb, fields = transition(m, down, at)
	if hb.Status != model.MonitorDown || !hb.Important || fields["last_status"] != model.MonitorDown {
		t.Fatalf("third failure should flip to down: %+v %v", hb, fields)
	}
	// Already down: further failures are down but not important.
	hb, _ = transition(m, down, at)
	if hb.Status != model.MonitorDown || hb.Important {
		t.Fatalf("repeat down: %+v", hb)
	}
	// Recovery is important and resets failures.
	exp := at.Add(48 * time.Hour)
	hb, fields = transition(m, Result{Status: model.MonitorUp, HTTPStatus: 200, CertExpiresAt: &exp}, at)
	if hb.Status != model.MonitorUp || !hb.Important || fields["failure_count"] != 0 || fields["cert_expires_at"] != exp {
		t.Fatalf("recovery: %+v %v", hb, fields)
	}
}

func TestTransitionNewMonitorNoRetries(t *testing.T) {
	m := &model.Monitor{Base: model.Base{ID: uuid.New()}}
	hb, _ := transition(m, Result{Status: model.MonitorDown}, time.Now())
	if hb.Status != model.MonitorDown || !hb.Important {
		t.Fatalf("pending -> down should be important: %+v", hb)
	}
	m2 := &model.Monitor{Base: model.Base{ID: uuid.New()}}
	hb, _ = transition(m2, Result{Status: model.MonitorUp}, time.Now())
	if hb.Status != model.MonitorUp || !hb.Important {
		t.Fatalf("pending -> up should be important: %+v", hb)
	}
}

type recordingSender struct {
	mu     sync.Mutex
	events []Event
}

func (r *recordingSender) Send(_ context.Context, _ model.NotificationConfig, ev Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, ev)
	return nil
}

func TestWorkerPublishesAndNotifiesOnChange(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	m := *baseMonitor(srv.URL)
	m.ID = uuid.New()
	m.Active = true
	m.Retries = 1
	m.Notifications = []model.MonitorNotification{
		{Base: model.Base{ID: uuid.New()}, MonitorID: m.ID, Type: model.NotificationWebhook, Enabled: true, Config: []byte(`{"post_url":"https://hooks.example/x"}`)},
		{Base: model.Base{ID: uuid.New()}, MonitorID: m.ID, Type: model.NotificationWebhook, Enabled: false, Config: []byte(`{"post_url":"https://hooks.example/disabled"}`)},
	}
	store := newFakeStore(m)
	broker := messaging.NewMemory()
	rec := &recordingSender{}
	disp := NewDispatcherWithSenders(slog.Default(), map[model.NotificationType]Sender{model.NotificationWebhook: rec})
	w := NewWorker(store, localProber(), broker, disp, slog.Default(), WorkerOptions{Tick: time.Hour, Concurrency: 4})
	ctx := context.Background()

	w.RunOnce(ctx) // pending -> up (important)
	w.RunOnce(ctx) // not due yet
	if len(store.heartbeats) != 1 {
		t.Fatalf("second RunOnce should skip a monitor that is not due; heartbeats=%d", len(store.heartbeats))
	}
	store.monitors[0].NextCheckAt = nil
	fail.Store(true)
	w.RunOnce(ctx) // retry 1/1 -> pending
	store.monitors[0].NextCheckAt = nil
	w.RunOnce(ctx) // down (important)
	store.monitors[0].NextCheckAt = nil
	fail.Store(false)
	w.RunOnce(ctx) // up (important)

	want := []string{messaging.SubjectMonitorUp, messaging.SubjectMonitorDown, messaging.SubjectMonitorUp}
	got := broker.Subjects()
	if len(got) != len(want) {
		t.Fatalf("published %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published %v want %v", got, want)
		}
	}
	var env messaging.Envelope
	if err := json.Unmarshal(broker.Published[1].Data, &env); err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data["status"] != "down" || data["http_status"] != float64(503) || data["name"] != "t" {
		t.Fatalf("down payload: %v", data)
	}

	statuses := []model.MonitorStatus{}
	for _, hb := range store.heartbeats {
		statuses = append(statuses, hb.Status)
	}
	wantHB := []model.MonitorStatus{model.MonitorUp, model.MonitorPending, model.MonitorDown, model.MonitorUp}
	if len(statuses) != len(wantHB) {
		t.Fatalf("heartbeats %v want %v", statuses, wantHB)
	}
	for i := range wantHB {
		if statuses[i] != wantHB[i] {
			t.Fatalf("heartbeats %v want %v", statuses, wantHB)
		}
	}
	if len(rec.events) != 3 {
		t.Fatalf("enabled webhook should receive 3 events (disabled one none), got %d", len(rec.events))
	}
	if rec.events[1].Status != model.MonitorDown || rec.events[1].Kind != KindStatus {
		t.Fatalf("notify payload: %+v", rec.events[1])
	}
}

func TestDispatcherHandlesUnsupportedAndInvalid(t *testing.T) {
	disp := NewDispatcherWithSenders(slog.Default(), map[model.NotificationType]Sender{
		model.NotificationWebPush: WebPushSender{},
	})
	notes := []model.MonitorNotification{
		{Base: model.Base{ID: uuid.New()}, Type: model.NotificationWebPush, Enabled: true, Config: []byte(`{"endpoint":"https://push.example/1","keys":{"p256dh":"a","auth":"b"}}`)},
		{Base: model.Base{ID: uuid.New()}, Type: model.NotificationWebPush, Enabled: true, Config: []byte(`{"endpoint":"nope"}`)},
		{Base: model.Base{ID: uuid.New()}, Type: model.NotificationDiscord, Enabled: true, Config: []byte(`{}`)},
	}
	// Must not panic or block; all three paths are log-only.
	disp.Dispatch(context.Background(), notes, Event{Name: "x", Kind: KindStatus, Status: model.MonitorDown})
	disp.Dispatch(context.Background(), notes, Event{Name: "x", Kind: KindStatus, Status: model.MonitorUp})
	if _, warned := disp.warned.Load(notes[0].ID); !warned {
		t.Fatal("unsupported channel should be recorded as warned")
	}
}
