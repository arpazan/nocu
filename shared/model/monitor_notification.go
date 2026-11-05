package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// NotificationType is the channel a MonitorNotification delivers to.
type NotificationType string

const (
	NotificationDiscord NotificationType = "discord"
	NotificationWebPush NotificationType = "webpush"
	NotificationWebhook NotificationType = "webhook"
)

// Valid reports whether the type is one of the known values.
func (t NotificationType) Valid() bool {
	switch t {
	case NotificationDiscord, NotificationWebPush, NotificationWebhook:
		return true
	}
	return false
}

// MonitorNotification is one delivery channel attached to a monitor.
// Config holds the channel-specific settings; DecodeNotificationConfig turns
// it into the matching typed struct.
type MonitorNotification struct {
	Base
	MonitorID uuid.UUID        `gorm:"type:uuid;not null;index" json:"monitor_id"`
	Type      NotificationType `gorm:"size:16;not null" json:"type"`
	Enabled   bool             `gorm:"not null;default:true" json:"enabled"`
	Config    datatypes.JSON   `gorm:"type:jsonb;not null" json:"config"`
}

// NotificationConfig is implemented by every typed channel config.
// Validate applies defaults in place and reports the first problem found.
type NotificationConfig interface {
	Validate() error
}

// DiscordMessageFormat selects between a plain content message and a rich embed.
type DiscordMessageFormat string

const (
	DiscordFormatText  DiscordMessageFormat = "text"
	DiscordFormatEmbed DiscordMessageFormat = "embed"
)

// Valid reports whether the format is one of the known values.
func (f DiscordMessageFormat) Valid() bool {
	return f == DiscordFormatText || f == DiscordFormatEmbed
}

// DiscordConfig configures a Discord webhook channel.
type DiscordConfig struct {
	WebhookURL          string               `json:"webhook_url"`
	BotDisplayName      string               `json:"bot_display_name,omitempty"`
	PrefixCustomMessage string               `json:"prefix_custom_message,omitempty"`
	MessageFormat       DiscordMessageFormat `json:"message_format"`
	DisableURL          bool                 `json:"disable_url"`
}

// Validate implements NotificationConfig.
func (c *DiscordConfig) Validate() error {
	u, err := url.Parse(strings.TrimSpace(c.WebhookURL))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("discord: webhook_url must be an https URL")
	}
	host := strings.ToLower(u.Hostname())
	if !strings.HasSuffix(host, "discord.com") && !strings.HasSuffix(host, "discordapp.com") {
		return errors.New("discord: webhook_url must point at discord.com")
	}
	if !strings.HasPrefix(u.Path, "/api/webhooks/") {
		return errors.New("discord: webhook_url must be a /api/webhooks/ URL")
	}
	c.WebhookURL = u.String()
	c.BotDisplayName = strings.TrimSpace(c.BotDisplayName)
	c.PrefixCustomMessage = strings.TrimSpace(c.PrefixCustomMessage)
	if c.MessageFormat == "" {
		c.MessageFormat = DiscordFormatText
	}
	if !c.MessageFormat.Valid() {
		return errors.New("discord: message_format must be text or embed")
	}
	return nil
}

// WebPushKeys are the client keys of a browser PushSubscription.
type WebPushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// WebPushConfig stores a browser PushSubscription (PushSubscription.toJSON()).
type WebPushConfig struct {
	Endpoint string      `json:"endpoint"`
	Keys     WebPushKeys `json:"keys"`
}

// Validate implements NotificationConfig.
func (c *WebPushConfig) Validate() error {
	u, err := url.Parse(strings.TrimSpace(c.Endpoint))
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return errors.New("webpush: endpoint must be an https URL")
	}
	c.Endpoint = u.String()
	c.Keys.P256dh = strings.TrimSpace(c.Keys.P256dh)
	c.Keys.Auth = strings.TrimSpace(c.Keys.Auth)
	if c.Keys.P256dh == "" || c.Keys.Auth == "" {
		return errors.New("webpush: keys.p256dh and keys.auth are required")
	}
	return nil
}

// WebhookBodyType selects how a webhook channel encodes its payload.
type WebhookBodyType string

const (
	WebhookBodyJSON   WebhookBodyType = "json"
	WebhookBodyForm   WebhookBodyType = "form"
	WebhookBodyCustom WebhookBodyType = "custom"
)

// Valid reports whether the body type is one of the known values.
func (b WebhookBodyType) Valid() bool {
	switch b {
	case WebhookBodyJSON, WebhookBodyForm, WebhookBodyCustom:
		return true
	}
	return false
}

// WebhookConfig configures a generic HTTP webhook channel.
//
// CustomBody is a template used when RequestBody is custom. Placeholders
// {{monitor_name}}, {{monitor_url}}, {{status}} and {{message}} are replaced
// at delivery time.
type WebhookConfig struct {
	PostURL           string            `json:"post_url"`
	HTTPMethod        HTTPMethod        `json:"http_method"`
	RequestBody       WebhookBodyType   `json:"request_body"`
	CustomBody        string            `json:"custom_body,omitempty"`
	AdditionalHeaders map[string]string `json:"additional_headers,omitempty"`
}

// Validate implements NotificationConfig.
func (c *WebhookConfig) Validate() error {
	u, err := url.Parse(strings.TrimSpace(c.PostURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("webhook: post_url must be an http or https URL")
	}
	c.PostURL = u.String()
	if c.HTTPMethod == "" {
		c.HTTPMethod = MethodPOST
	}
	switch c.HTTPMethod {
	case MethodPOST, MethodPUT, MethodPATCH:
	default:
		return errors.New("webhook: http_method must be POST, PUT or PATCH")
	}
	if c.RequestBody == "" {
		c.RequestBody = WebhookBodyJSON
	}
	if !c.RequestBody.Valid() {
		return errors.New("webhook: request_body must be json, form or custom")
	}
	if c.RequestBody == WebhookBodyCustom && strings.TrimSpace(c.CustomBody) == "" {
		return errors.New("webhook: custom_body is required when request_body is custom")
	}
	for k := range c.AdditionalHeaders {
		if strings.TrimSpace(k) == "" {
			return errors.New("webhook: additional_headers keys cannot be empty")
		}
	}
	return nil
}

// DecodeNotificationConfig parses raw JSON into the typed config for t.
// Unknown fields are rejected so typos surface at save time.
func DecodeNotificationConfig(t NotificationType, raw []byte) (NotificationConfig, error) {
	var cfg NotificationConfig
	switch t {
	case NotificationDiscord:
		cfg = &DiscordConfig{}
	case NotificationWebPush:
		cfg = &WebPushConfig{}
	case NotificationWebhook:
		cfg = &WebhookConfig{}
	default:
		return nil, fmt.Errorf("unknown notification type %q", t)
	}
	if len(raw) == 0 {
		return nil, errors.New("config is required")
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// Decode returns the typed config of this notification.
func (n *MonitorNotification) Decode() (NotificationConfig, error) {
	return DecodeNotificationConfig(n.Type, n.Config)
}
