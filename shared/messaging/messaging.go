// Package messaging defines the publish/subscribe contract between services
// and provides a NATS JetStream implementation plus an in-memory one for tests.
package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Stream and subject layout. Every subject lives under one JetStream stream.
const (
	StreamName = "NOCU"

	SubjectIncidentAll         = "incident.>"
	SubjectIncidentCreated     = "incident.created"
	SubjectIncidentUpdated     = "incident.updated"
	SubjectIncidentDeleted     = "incident.deleted"
	SubjectIncidentSLABreached = "incident.sla_breached"

	SubjectJobAll       = "job.>"
	SubjectJobHeartbeat = "job.heartbeat"
	SubjectJobCompleted = "job.completed"
	SubjectJobFailed    = "job.failed"

	SubjectMonitorAll            = "monitor.>"
	SubjectMonitorCreated        = "monitor.created"
	SubjectMonitorUpdated        = "monitor.updated"
	SubjectMonitorDeleted        = "monitor.deleted"
	SubjectMonitorUp             = "monitor.up"
	SubjectMonitorDown           = "monitor.down"
	SubjectMonitorCertExpiring   = "monitor.cert_expiring"
	SubjectMonitorDomainExpiring = "monitor.domain_expiring"
)

// StreamSubjects lists the wildcard subjects the stream captures.
var StreamSubjects = []string{SubjectIncidentAll, SubjectJobAll, SubjectMonitorAll}

// Message is a received message.
type Message struct {
	Subject string
	Data    []byte
	// Deliveries is how many times the broker has delivered this message (1 on first delivery).
	Deliveries uint64
}

// Handler processes a message. Returning an error asks the broker to redeliver.
type Handler func(ctx context.Context, msg Message) error

// Subscription is a running subscription.
type Subscription interface {
	Unsubscribe() error
}

// Publisher sends messages.
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// Subscriber receives messages.
//
// durable == "" creates an ephemeral, at-most-once subscription (fan-out to
// every running instance, used by the WebSocket hub). A durable name creates
// a JetStream consumer with explicit acks shared by all instances with that
// name (work-queue semantics, used by the consumer service).
type Subscriber interface {
	Subscribe(ctx context.Context, subject, durable string, h Handler) (Subscription, error)
}

// Client is the full broker interface.
type Client interface {
	Publisher
	Subscriber
	Ping(ctx context.Context) error
	Close() error
}

// Envelope is the JSON body of every message published by this monorepo.
type Envelope struct {
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	Source     string          `json:"source"`
	Data       json.RawMessage `json:"data"`
}

// PublishJSON wraps v in an Envelope and publishes it.
func PublishJSON(ctx context.Context, p Publisher, subject, source string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("messaging: marshal data: %w", err)
	}
	env := Envelope{Type: subject, OccurredAt: time.Now().UTC(), Source: source, Data: data}
	body, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("messaging: marshal envelope: %w", err)
	}
	return p.Publish(ctx, subject, body)
}
