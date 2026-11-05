package ws

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
)

func TestBroadcastDeliversToEveryClient(t *testing.T) {
	hub := NewHub(slog.Default(), 4)
	a := hub.Register(uuid.New())
	b := hub.Register(uuid.New())

	hub.Broadcast([]byte("one"))

	for _, c := range []*Client{a, b} {
		select {
		case got := <-c.send:
			if string(got) != "one" {
				t.Fatalf("got %q", got)
			}
		default:
			t.Fatal("client did not receive broadcast")
		}
	}
	if hub.Count() != 2 {
		t.Fatalf("count = %d", hub.Count())
	}
}

func TestBroadcastDropsSlowClient(t *testing.T) {
	hub := NewHub(slog.Default(), 1)
	slow := hub.Register(uuid.New())
	fast := hub.Register(uuid.New())

	hub.Broadcast([]byte("1"))
	<-fast.send // fast drains, slow does not
	hub.Broadcast([]byte("2"))

	if hub.Count() != 1 {
		t.Fatalf("expected slow client dropped, count = %d", hub.Count())
	}
	if _, open := <-slow.send; open {
		// first message is still queued; next receive must observe closed channel
		if _, open := <-slow.send; open {
			t.Fatal("slow client queue not closed")
		}
	}
}

func TestUnregisterIsIdempotent(t *testing.T) {
	hub := NewHub(slog.Default(), 1)
	c := hub.Register(uuid.New())
	hub.Unregister(c)
	hub.Unregister(c)
	if hub.Count() != 0 {
		t.Fatal("client still registered")
	}
}

func TestBridgeForwardsBrokerMessages(t *testing.T) {
	hub := NewHub(slog.Default(), 4)
	c := hub.Register(uuid.New())
	broker := messaging.NewMemory()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Bridge(ctx, broker, hub, messaging.SubjectIncidentAll) }()

	// Wait until the subscription exists.
	for i := 0; i < 100; i++ {
		if err := messaging.PublishJSON(context.Background(), broker, messaging.SubjectIncidentCreated, "t", map[string]int{"n": i}); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-c.send:
			want := `"subject":"incident.created"`
			if !contains(string(got), want) {
				t.Fatalf("frame %s missing %s", got, want)
			}
			cancel()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			return
		default:
		}
	}
	t.Fatal("bridge never delivered a frame")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
