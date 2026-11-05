package uptime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Kind says what an Event is about.
type Kind string

const (
	KindStatus         Kind = "status"
	KindCertExpiring   Kind = "cert_expiring"
	KindDomainExpiring Kind = "domain_expiring"
)

// Event is what a notification channel receives.
type Event struct {
	MonitorID  uuid.UUID
	Name       string
	URL        string
	Status     model.MonitorStatus
	Message    string
	HTTPStatus int
	LatencyMS  int64
	CheckedAt  time.Time
	Kind       Kind
	// ExpiresAt and DaysLeft are set for cert_expiring and domain_expiring events.
	ExpiresAt *time.Time
	DaysLeft  int
}

// Title is the one-line human summary of the event.
func (e Event) Title() string {
	switch e.Kind {
	case KindCertExpiring:
		return fmt.Sprintf("%s: TLS certificate expires in %d days", e.Name, e.DaysLeft)
	case KindDomainExpiring:
		return fmt.Sprintf("%s: domain expires in %d days", e.Name, e.DaysLeft)
	}
	return fmt.Sprintf("%s is %s", e.Name, strings.ToUpper(string(e.Status)))
}

// Fields is the flat payload used by webhook bodies.
func (e Event) Fields() map[string]any {
	out := map[string]any{
		"monitor_id":   e.MonitorID,
		"monitor_name": e.Name,
		"monitor_url":  e.URL,
		"status":       e.Status,
		"message":      e.Message,
		"http_status":  e.HTTPStatus,
		"latency_ms":   e.LatencyMS,
		"checked_at":   e.CheckedAt,
		"kind":         e.Kind,
	}
	if e.ExpiresAt != nil {
		out["expires_at"] = *e.ExpiresAt
		out["days_left"] = e.DaysLeft
	}
	return out
}

// Sender delivers an Event over one channel type.
type Sender interface {
	Send(ctx context.Context, cfg model.NotificationConfig, ev Event) error
}

// Dispatcher fans an Event out to a monitor's enabled notifications.
// Delivery is best-effort: failures are logged, never returned, so a broken
// webhook cannot stall probing.
type Dispatcher struct {
	senders map[model.NotificationType]Sender
	log     *slog.Logger
	timeout time.Duration
	warned  sync.Map // notification id -> struct{} for unsupported channels
}

// NewDispatcher builds a Dispatcher with the built-in senders sharing one guarded client.
func NewDispatcher(log *slog.Logger, allowPrivate bool) *Dispatcher {
	client := NewHTTPClient(allowPrivate)
	return NewDispatcherWithSenders(log, map[model.NotificationType]Sender{
		model.NotificationDiscord: &DiscordSender{Client: client},
		model.NotificationWebhook: &WebhookSender{Client: client},
		model.NotificationWebPush: &WebPushSender{},
	})
}

// NewDispatcherWithSenders builds a Dispatcher from explicit senders (tests).
func NewDispatcherWithSenders(log *slog.Logger, senders map[model.NotificationType]Sender) *Dispatcher {
	return &Dispatcher{senders: senders, log: log.With("component", "notify"), timeout: 10 * time.Second}
}

// Dispatch sends ev to every enabled notification in notes.
func (d *Dispatcher) Dispatch(ctx context.Context, notes []model.MonitorNotification, ev Event) {
	for i := range notes {
		n := &notes[i]
		if !n.Enabled {
			continue
		}
		log := d.log.With("monitor_id", ev.MonitorID, "notification_id", n.ID, "type", n.Type, "kind", ev.Kind)
		sender, ok := d.senders[n.Type]
		if !ok {
			log.WarnContext(ctx, "no sender for notification type")
			continue
		}
		cfg, err := n.Decode()
		if err != nil {
			log.ErrorContext(ctx, "notification config invalid", "err", err)
			continue
		}
		if err := cfg.Validate(); err != nil {
			log.ErrorContext(ctx, "notification config invalid", "err", err)
			continue
		}

		sctx, cancel := context.WithTimeout(ctx, d.timeout)
		err = sender.Send(sctx, cfg, ev)
		cancel()
		switch {
		case errors.Is(err, ErrNotSupported):
			if _, seen := d.warned.LoadOrStore(n.ID, struct{}{}); !seen {
				log.WarnContext(ctx, "notification channel not supported yet", "err", err)
			}
		case err != nil:
			log.ErrorContext(ctx, "notification failed", "err", err)
		default:
			log.InfoContext(ctx, "notification sent")
		}
	}
}

// doRequest sends req with client and treats any non-2xx as an error. The
// response body is read (bounded) so a short diagnostic can be included.
func doRequest(ctx context.Context, client *http.Client, method, target string, body io.Reader, headers map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	for k, v := range headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %s: %s", resp.Status, strings.TrimSpace(string(snippet)))
	}
	return nil
}
