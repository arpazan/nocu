package uptime

import (
	"context"
	"errors"

	"github.com/Pzquzer/Nocu/shared/model"
)

// ErrNotSupported is returned by channels that are stored and validated but
// not yet delivered.
var ErrNotSupported = errors.New("uptime: webpush delivery is not supported yet (needs VAPID keys and RFC 8291 payload encryption)")

// WebPushSender is a placeholder. Web Push requires a VAPID key pair in the
// service configuration and RFC 8291 (aes128gcm) payload encryption, neither
// of which this milestone ships. Subscriptions are accepted and validated by
// core-api so clients can register now; delivery lands in a later milestone.
type WebPushSender struct{}

// Send implements Sender and always returns ErrNotSupported.
func (WebPushSender) Send(context.Context, model.NotificationConfig, Event) error {
	return ErrNotSupported
}
