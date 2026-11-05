package messaging

import (
	"context"
	"strings"
	"sync"
)

// Memory is an in-process Client for tests. Publish delivers synchronously to
// every matching subscriber and records the message.
type Memory struct {
	mu        sync.Mutex
	subs      []*memorySub
	Published []Message
}

type memorySub struct {
	subject string
	h       Handler
	owner   *Memory
}

// NewMemory returns an empty in-memory broker.
func NewMemory() *Memory { return &Memory{} }

func (m *Memory) Publish(ctx context.Context, subject string, data []byte) error {
	m.mu.Lock()
	msg := Message{Subject: subject, Data: data, Deliveries: 1}
	m.Published = append(m.Published, msg)
	subs := append([]*memorySub(nil), m.subs...)
	m.mu.Unlock()

	for _, s := range subs {
		if subjectMatches(s.subject, subject) {
			if err := s.h(ctx, msg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Memory) Subscribe(_ context.Context, subject, _ string, h Handler) (Subscription, error) {
	s := &memorySub{subject: subject, h: h, owner: m}
	m.mu.Lock()
	m.subs = append(m.subs, s)
	m.mu.Unlock()
	return s, nil
}

func (s *memorySub) Unsubscribe() error {
	s.owner.mu.Lock()
	defer s.owner.mu.Unlock()
	for i, cur := range s.owner.subs {
		if cur == s {
			s.owner.subs = append(s.owner.subs[:i], s.owner.subs[i+1:]...)
			break
		}
	}
	return nil
}

func (m *Memory) Ping(context.Context) error { return nil }
func (m *Memory) Close() error               { return nil }

// Subjects returns the subjects published so far, in order.
func (m *Memory) Subjects() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.Published))
	for _, p := range m.Published {
		out = append(out, p.Subject)
	}
	return out
}

// subjectMatches implements NATS wildcard rules: "*" matches one token, ">" matches the rest.
func subjectMatches(pattern, subject string) bool {
	p := strings.Split(pattern, ".")
	s := strings.Split(subject, ".")
	for i, tok := range p {
		if tok == ">" {
			return len(s) >= i+1
		}
		if i >= len(s) {
			return false
		}
		if tok != "*" && tok != s[i] {
			return false
		}
	}
	return len(p) == len(s)
}
