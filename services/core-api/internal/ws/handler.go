package ws

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"

	"github.com/Pzquzer/Nocu/shared/httpx"
	"github.com/Pzquzer/Nocu/shared/messaging"
)

// Options tunes the connection handler.
type Options struct {
	WriteTimeout time.Duration
	PingInterval time.Duration
	// OriginPatterns is passed to the upgrader. "*" allows any origin.
	OriginPatterns []string
}

func (o *Options) defaults() {
	if o.WriteTimeout <= 0 {
		o.WriteTimeout = 5 * time.Second
	}
	if o.PingInterval <= 0 {
		o.PingInterval = 30 * time.Second
	}
}

// Handler upgrades HTTP requests and runs the connection pumps.
type Handler struct {
	hub  *Hub
	opts Options
}

// NewHandler builds a Handler.
func NewHandler(hub *Hub, opts Options) *Handler {
	opts.defaults()
	return &Handler{hub: hub, opts: opts}
}

// Register mounts GET /ws on an authenticated group.
func (h *Handler) Register(g *echo.Group) {
	g.GET("/ws", h.serve)
}

func (h *Handler) serve(c echo.Context) error {
	claims, ok := httpx.ClaimsFrom(c)
	if !ok {
		return httpx.Unauthorized("missing credentials")
	}

	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		OriginPatterns: h.opts.OriginPatterns,
	})
	if err != nil {
		// Accept already wrote an HTTP error response.
		return nil
	}

	client := h.hub.Register(claims.UserID)
	defer h.hub.Unregister(client)

	// CloseRead drains inbound frames (handling ping/pong and close) and
	// cancels ctx when the peer goes away.
	ctx := conn.CloseRead(c.Request().Context())

	// Greeting lets clients confirm the session is live.
	hello, _ := json.Marshal(map[string]any{"type": "hello", "user_id": claims.UserID, "at": time.Now().UTC()})
	if err := h.write(ctx, conn, hello); err != nil {
		return nil
	}

	ticker := time.NewTicker(h.opts.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = conn.Close(websocket.StatusNormalClosure, "bye")
			return nil
		case msg, open := <-client.send:
			if !open {
				_ = conn.Close(websocket.StatusTryAgainLater, "server dropped connection")
				return nil
			}
			if err := h.write(ctx, conn, msg); err != nil {
				return nil
			}
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, h.opts.WriteTimeout)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				_ = conn.Close(websocket.StatusPolicyViolation, "ping timeout")
				return nil
			}
		}
	}
}

func (h *Handler) write(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	wctx, cancel := context.WithTimeout(ctx, h.opts.WriteTimeout)
	defer cancel()
	return conn.Write(wctx, websocket.MessageText, msg)
}

// Bridge subscribes the hub to broker subjects so every published event is
// pushed to every connected client. It returns when ctx is done.
func Bridge(ctx context.Context, sub messaging.Subscriber, hub *Hub, subjects ...string) error {
	subs := make([]messaging.Subscription, 0, len(subjects))
	for _, subject := range subjects {
		s, err := sub.Subscribe(ctx, subject, "", func(_ context.Context, msg messaging.Message) error {
			hub.Broadcast(frame(msg))
			return nil
		})
		if err != nil {
			for _, done := range subs {
				_ = done.Unsubscribe()
			}
			return err
		}
		subs = append(subs, s)
	}
	<-ctx.Done()
	var errs []error
	for _, s := range subs {
		errs = append(errs, s.Unsubscribe())
	}
	return errors.Join(errs...)
}

// frame is the wire format pushed to clients: the broker subject plus the raw envelope.
func frame(msg messaging.Message) []byte {
	payload := json.RawMessage(msg.Data)
	if !json.Valid(payload) {
		payload, _ = json.Marshal(string(msg.Data))
	}
	out, _ := json.Marshal(map[string]any{
		"type":    "event",
		"subject": msg.Subject,
		"payload": payload,
	})
	return out
}
