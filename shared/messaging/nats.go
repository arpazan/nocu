package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATS implements Client on top of NATS JetStream.
type NATS struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	stream jetstream.Stream
	log    *slog.Logger
}

// NATSOptions configures ConnectNATS.
type NATSOptions struct {
	URL    string
	Name   string
	Logger *slog.Logger
	// MaxAge bounds how long messages stay in the stream.
	MaxAge time.Duration
}

// ConnectNATS dials the server and ensures the shared stream exists.
func ConnectNATS(ctx context.Context, opts NATSOptions) (*NATS, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.MaxAge == 0 {
		opts.MaxAge = 24 * time.Hour
	}
	log := opts.Logger.With("component", "nats")

	nc, err := nats.Connect(opts.URL,
		nats.Name(opts.Name),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				log.Warn("disconnected", "err", err)
			}
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			log.Info("reconnected", "url", c.ConnectedUrl())
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("messaging: connect: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("messaging: jetstream: %w", err)
	}

	stream, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      StreamName,
		Subjects:  StreamSubjects,
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
		MaxAge:    opts.MaxAge,
	})
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("messaging: ensure stream: %w", err)
	}

	return &NATS{nc: nc, js: js, stream: stream, log: log}, nil
}

// Publish sends a message through JetStream and waits for the ack.
func (n *NATS) Publish(ctx context.Context, subject string, data []byte) error {
	if _, err := n.js.Publish(ctx, subject, data); err != nil {
		return fmt.Errorf("messaging: publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe implements Subscriber. See the interface docs for durable semantics.
func (n *NATS) Subscribe(ctx context.Context, subject, durable string, h Handler) (Subscription, error) {
	if durable == "" {
		return n.subscribeCore(ctx, subject, h)
	}
	return n.subscribeDurable(ctx, subject, durable, h)
}

func (n *NATS) subscribeCore(ctx context.Context, subject string, h Handler) (Subscription, error) {
	sub, err := n.nc.Subscribe(subject, func(m *nats.Msg) {
		if err := h(ctx, Message{Subject: m.Subject, Data: m.Data, Deliveries: 1}); err != nil {
			n.log.Warn("handler failed", "subject", m.Subject, "err", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("messaging: subscribe %s: %w", subject, err)
	}
	return subscription{stop: sub.Unsubscribe}, nil
}

func (n *NATS) subscribeDurable(ctx context.Context, subject, durable string, h Handler) (Subscription, error) {
	cons, err := n.stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("messaging: consumer %s: %w", durable, err)
	}

	cc, err := cons.Consume(func(m jetstream.Msg) {
		deliveries := uint64(1)
		if meta, err := m.Metadata(); err == nil {
			deliveries = meta.NumDelivered
		}
		if err := h(ctx, Message{Subject: m.Subject(), Data: m.Data(), Deliveries: deliveries}); err != nil {
			n.log.Warn("handler failed, nak", "subject", m.Subject(), "deliveries", deliveries, "err", err)
			_ = m.NakWithDelay(backoff(deliveries))
			return
		}
		if err := m.Ack(); err != nil {
			n.log.Warn("ack failed", "subject", m.Subject(), "err", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("messaging: consume %s: %w", durable, err)
	}
	return subscription{stop: func() error { cc.Stop(); return nil }}, nil
}

func backoff(deliveries uint64) time.Duration {
	d := time.Second << min(deliveries, 5)
	return d
}

// Ping reports whether the connection is up.
func (n *NATS) Ping(_ context.Context) error {
	if !n.nc.IsConnected() {
		return errors.New("nats: not connected")
	}
	return nil
}

// Close drains the connection.
func (n *NATS) Close() error {
	return n.nc.Drain()
}

type subscription struct{ stop func() error }

func (s subscription) Unsubscribe() error { return s.stop() }
