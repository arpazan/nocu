package consumer

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

type fakeRepo struct {
	events []*model.Event
	err    error
}

func (f *fakeRepo) Create(_ context.Context, ev *model.Event) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, ev)
	return nil
}

func TestHandleStoresEvent(t *testing.T) {
	repo := &fakeRepo{}
	h := NewHandler(repo, slog.Default())

	err := h.Handle(context.Background(), messaging.Message{Subject: messaging.SubjectIncidentCreated, Data: []byte(`{"a":1}`)})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.events) != 1 {
		t.Fatalf("stored %d events", len(repo.events))
	}
	ev := repo.events[0]
	if ev.Type != "created" || ev.Subject != messaging.SubjectIncidentCreated || string(ev.Payload) != `{"a":1}` {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestHandleWrapsInvalidJSON(t *testing.T) {
	repo := &fakeRepo{}
	h := NewHandler(repo, slog.Default())
	if err := h.Handle(context.Background(), messaging.Message{Subject: "job.failed", Data: []byte("not json")}); err != nil {
		t.Fatal(err)
	}
	if got := string(repo.events[0].Payload); got != `{"raw":"not json"}` {
		t.Fatalf("payload = %s", got)
	}
	if err := h.Handle(context.Background(), messaging.Message{Subject: "job.heartbeat"}); err != nil {
		t.Fatal(err)
	}
	if got := string(repo.events[1].Payload); got != `{}` {
		t.Fatalf("empty payload = %s", got)
	}
}

func TestHandlePropagatesRepoError(t *testing.T) {
	boom := errors.New("db down")
	h := NewHandler(&fakeRepo{err: boom}, slog.Default())
	if err := h.Handle(context.Background(), messaging.Message{Subject: "x.y"}); !errors.Is(err, boom) {
		t.Fatalf("expected repo error, got %v", err)
	}
}
