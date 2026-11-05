package uptime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Discord embed colours.
const (
	colorUp      = 0x2ECC71
	colorDown    = 0xE74C3C
	colorWarning = 0xE67E22
)

// DiscordSender posts to a Discord webhook.
type DiscordSender struct{ Client *http.Client }

// Send implements Sender.
func (s *DiscordSender) Send(ctx context.Context, cfg model.NotificationConfig, ev Event) error {
	c, ok := cfg.(*model.DiscordConfig)
	if !ok {
		return errors.New("discord: wrong config type")
	}
	body, err := json.Marshal(discordPayload(c, ev))
	if err != nil {
		return err
	}
	return doRequest(ctx, s.Client, http.MethodPost, c.WebhookURL, bytes.NewReader(body),
		map[string]string{"Content-Type": "application/json"})
}

// discordPayload builds the webhook body for either message format.
func discordPayload(c *model.DiscordConfig, ev Event) map[string]any {
	msg := map[string]any{}
	if c.BotDisplayName != "" {
		msg["username"] = c.BotDisplayName
	}

	if c.MessageFormat == model.DiscordFormatEmbed {
		fields := []map[string]any{
			{"name": "Status", "value": statusLabel(ev), "inline": true},
		}
		if !c.DisableURL {
			fields = append(fields, map[string]any{"name": "URL", "value": ev.URL, "inline": true})
		}
		if ev.Message != "" {
			fields = append(fields, map[string]any{"name": "Message", "value": ev.Message})
		}
		if ev.ExpiresAt != nil {
			fields = append(fields, map[string]any{"name": "Expires at", "value": ev.ExpiresAt.UTC().Format(time.RFC3339)})
		}
		fields = append(fields, map[string]any{"name": "Checked at", "value": ev.CheckedAt.UTC().Format(time.RFC3339)})

		embed := map[string]any{
			"title":     ev.Title(),
			"color":     colorFor(ev),
			"fields":    fields,
			"timestamp": ev.CheckedAt.UTC().Format(time.RFC3339),
		}
		if c.PrefixCustomMessage != "" {
			embed["description"] = c.PrefixCustomMessage
		}
		msg["embeds"] = []any{embed}
		return msg
	}

	var b strings.Builder
	if c.PrefixCustomMessage != "" {
		b.WriteString(c.PrefixCustomMessage)
		b.WriteString("\n")
	}
	b.WriteString(ev.Title())
	if !c.DisableURL {
		b.WriteString(" (" + ev.URL + ")")
	}
	if ev.Message != "" {
		b.WriteString("\n" + ev.Message)
	}
	msg["content"] = b.String()
	return msg
}

func statusLabel(ev Event) string {
	switch ev.Kind {
	case KindCertExpiring:
		return "certificate expiring"
	case KindDomainExpiring:
		return "domain expiring"
	}
	return strings.ToUpper(string(ev.Status))
}

func colorFor(ev Event) int {
	switch {
	case ev.Kind != KindStatus:
		return colorWarning
	case ev.Status == model.MonitorUp:
		return colorUp
	default:
		return colorDown
	}
}
