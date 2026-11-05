package consumer

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

// Handler turns a broker message into an Event row.
type Handler struct {
	repo EventRepository
	log  *slog.Logger
	now  func() time.Time
}

// NewHandler builds a Handler.
func NewHandler(repo EventRepository, log *slog.Logger) *Handler {
	return &Handler{repo: repo, log: log.With("component", "consumer"), now: func() time.Time { return time.Now().UTC() }}
}

// Handle implements messaging.Handler. A returned error triggers redelivery,
// so only transient failures (the database write) return one; malformed
// payloads are stored verbatim instead of poisoning the queue.
func (h *Handler) Handle(ctx context.Context, msg messaging.Message) error {
	ev := &model.Event{
		Subject:    msg.Subject,
		Type:       eventType(msg.Subject),
		Payload:    normalisePayload(msg.Data),
		ReceivedAt: h.now(),
	}
	if err := h.repo.Create(ctx, ev); err != nil {
		h.log.ErrorContext(ctx, "persist failed", "subject", msg.Subject, "deliveries", msg.Deliveries, "err", err)
		return err
	}
	h.log.InfoContext(ctx, "event stored", "subject", msg.Subject, "event_id", ev.ID)
	return nil
}

// eventType is the subject with its stream prefix removed: incident.created -> created.
func eventType(subject string) string {
	if i := strings.Index(subject, "."); i >= 0 {
		return subject[i+1:]
	}
	return subject
}

// normalisePayload guarantees a valid JSON document for the jsonb column.
func normalisePayload(data []byte) datatypes.JSON {
	if len(data) == 0 {
		return datatypes.JSON(`{}`)
	}
	if json.Valid(data) {
		return datatypes.JSON(data)
	}
	wrapped, _ := json.Marshal(map[string]string{"raw": string(data)})
	return datatypes.JSON(wrapped)
}
