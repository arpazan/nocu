package uptime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/Pzquzer/Nocu/shared/model"
)

// WebhookSender delivers to a generic HTTP endpoint.
type WebhookSender struct{ Client *http.Client }

// Send implements Sender.
func (s *WebhookSender) Send(ctx context.Context, cfg model.NotificationConfig, ev Event) error {
	c, ok := cfg.(*model.WebhookConfig)
	if !ok {
		return errors.New("webhook: wrong config type")
	}
	body, contentType, err := webhookBody(c, ev)
	if err != nil {
		return err
	}
	headers := map[string]string{"Content-Type": contentType}
	for k, v := range c.AdditionalHeaders {
		headers[k] = v // user headers win, including Content-Type
	}
	return doRequest(ctx, s.Client, string(c.HTTPMethod), c.PostURL, body, headers)
}

// webhookBody encodes the event per the configured request body type.
func webhookBody(c *model.WebhookConfig, ev Event) (io.Reader, string, error) {
	switch c.RequestBody {
	case model.WebhookBodyForm:
		v := url.Values{}
		fields := ev.Fields()
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v.Set(k, formValue(fields[k]))
		}
		return strings.NewReader(v.Encode()), "application/x-www-form-urlencoded", nil
	case model.WebhookBodyCustom:
		// A custom body is almost always JSON; callers override via additional_headers.
		return strings.NewReader(renderTemplate(c.CustomBody, ev)), "application/json", nil
	default:
		b, err := json.Marshal(ev.Fields())
		if err != nil {
			return nil, "", err
		}
		return bytes.NewReader(b), "application/json", nil
	}
}

func formValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	default:
		return fmt.Sprint(x)
	}
}

// renderTemplate replaces the supported placeholders in a custom body.
func renderTemplate(tpl string, ev Event) string {
	r := strings.NewReplacer(
		"{{monitor_name}}", ev.Name,
		"{{monitor_url}}", ev.URL,
		"{{status}}", string(ev.Status),
		"{{message}}", ev.Message,
	)
	return r.Replace(tpl)
}
