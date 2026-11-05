package uptime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

// RDAPClient looks up a domain's expiry date.
type RDAPClient interface {
	Expiry(ctx context.Context, domain string) (time.Time, error)
}

// ExpiryOptions tunes the ExpiryChecker.
type ExpiryOptions struct {
	// WarnDays lists the days-left thresholds that trigger a warning.
	WarnDays []int
	// RDAP overrides the lookup client (tests). Nil uses rdap.org.
	RDAP RDAPClient
}

// ExpiryChecker warns about expiring TLS certificates and domain registrations.
// It is run once a day by the cron Runner.
type ExpiryChecker struct {
	store    Store
	pub      messaging.Publisher
	dispatch *Dispatcher
	log      *slog.Logger
	warnDays []int
	rdap     RDAPClient
	now      func() time.Time
}

// NewExpiryChecker wires an ExpiryChecker.
func NewExpiryChecker(store Store, pub messaging.Publisher, dispatch *Dispatcher, log *slog.Logger, opts ExpiryOptions) *ExpiryChecker {
	rdap := opts.RDAP
	if rdap == nil {
		rdap = &rdapClient{client: &http.Client{Transport: newTransport(false, nil), Timeout: 15 * time.Second}, base: "https://rdap.org"}
	}
	days := append([]int(nil), opts.WarnDays...)
	sort.Ints(days)
	return &ExpiryChecker{
		store:    store,
		pub:      pub,
		dispatch: dispatch,
		log:      log.With("component", "expiry"),
		warnDays: days,
		rdap:     rdap,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// Run checks every candidate monitor. Per-monitor failures are logged and skipped.
func (c *ExpiryChecker) Run(ctx context.Context) error {
	monitors, err := c.store.ExpiryCandidates(ctx)
	if err != nil {
		return err
	}
	now := c.now()
	warned := 0
	for i := range monitors {
		m := &monitors[i]
		log := c.log.With("monitor_id", m.ID, "monitor", m.Name)

		if m.CertExpiryNotification {
			if m.CertExpiresAt == nil {
				log.Debug("no certificate expiry recorded yet")
			} else if c.evaluate(ctx, m, KindCertExpiring, *m.CertExpiresAt, now) {
				warned++
			}
		}

		if m.DomainExpiryNotification {
			host := hostOf(m.URL)
			if host == "" || net.ParseIP(host) != nil {
				log.Debug("skipping domain expiry: no hostname")
				continue
			}
			exp, err := c.rdap.Expiry(ctx, RegistrableDomain(host))
			if err != nil {
				log.WarnContext(ctx, "domain expiry lookup failed", "host", host, "err", err)
				continue
			}
			if err := c.store.UpdateMonitor(ctx, m.ID, map[string]any{"domain_expires_at": exp}); err != nil {
				log.ErrorContext(ctx, "store domain expiry failed", "err", err)
			}
			m.DomainExpiresAt = &exp
			if c.evaluate(ctx, m, KindDomainExpiring, exp, now) {
				warned++
			}
		}
	}
	c.log.Info("expiry check complete", "monitors", len(monitors), "warnings", warned)
	return nil
}

// evaluate publishes and notifies when expiresAt crosses a warning threshold.
func (c *ExpiryChecker) evaluate(ctx context.Context, m *model.Monitor, kind Kind, expiresAt time.Time, now time.Time) bool {
	days := DaysLeft(now, expiresAt)
	if !ShouldWarn(days, c.warnDays) {
		return false
	}
	exp := expiresAt.UTC()
	ev := Event{
		MonitorID: m.ID,
		Name:      m.Name,
		URL:       m.URL,
		Status:    m.LastStatus,
		CheckedAt: now,
		Kind:      kind,
		ExpiresAt: &exp,
		DaysLeft:  days,
	}
	ev.Message = ev.Title()

	subject := messaging.SubjectMonitorCertExpiring
	if kind == KindDomainExpiring {
		subject = messaging.SubjectMonitorDomainExpiring
	}
	if c.pub != nil {
		body := map[string]any{
			"monitor_id": m.ID,
			"name":       m.Name,
			"url":        m.URL,
			"expires_at": exp,
			"days_left":  days,
		}
		if err := messaging.PublishJSON(ctx, c.pub, subject, source, body); err != nil {
			c.log.ErrorContext(ctx, "publish failed", "subject", subject, "monitor_id", m.ID, "err", err)
		}
	}
	if c.dispatch != nil {
		c.dispatch.Dispatch(ctx, m.Notifications, ev)
	}
	return true
}

// DaysLeft is the whole days between now and expiresAt, negative once expired.
func DaysLeft(now, expiresAt time.Time) int {
	return int(expiresAt.Sub(now).Hours() / 24)
}

// ShouldWarn reports whether daysLeft hits a threshold: exactly equal to any
// configured value, or at or below the smallest one (so an expired or nearly
// expired item warns every day).
func ShouldWarn(daysLeft int, thresholds []int) bool {
	if len(thresholds) == 0 {
		return false
	}
	minimum := thresholds[0]
	for _, t := range thresholds {
		if t < minimum {
			minimum = t
		}
		if daysLeft == t {
			return true
		}
	}
	return daysLeft <= minimum
}

func hostOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}

// secondLevelTLDs lists registries where the registrable name has three
// labels. This is a small table, not the public suffix list; names under
// other multi-label suffixes fall back to the last two labels and the RDAP
// lookup will then fail with a logged warning.
var secondLevelTLDs = map[string]bool{
	"co.uk": true, "org.uk": true, "ac.uk": true, "gov.uk": true, "me.uk": true,
	"com.au": true, "net.au": true, "org.au": true,
	"co.th": true, "or.th": true, "ac.th": true, "go.th": true, "in.th": true, "net.th": true, "mi.th": true,
	"co.jp": true, "ne.jp": true, "or.jp": true,
	"co.nz": true, "co.kr": true, "co.in": true, "co.za": true,
	"com.br": true, "com.sg": true, "com.my": true, "com.tw": true, "com.hk": true, "com.cn": true, "com.mx": true,
}

// RegistrableDomain reduces a hostname to the name a registry tracks:
// "api.example.co.th" -> "example.co.th", "www.example.com" -> "example.com".
func RegistrableDomain(host string) string {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	labels := strings.Split(host, ".")
	if len(labels) <= 2 {
		return host
	}
	lastTwo := strings.Join(labels[len(labels)-2:], ".")
	if secondLevelTLDs[lastTwo] && len(labels) >= 3 {
		return strings.Join(labels[len(labels)-3:], ".")
	}
	return lastTwo
}

// rdapClient queries an RDAP bootstrap service such as rdap.org.
type rdapClient struct {
	client *http.Client
	base   string
}

// NewRDAPClient returns an RDAPClient against base (e.g. "https://rdap.org").
func NewRDAPClient(client *http.Client, base string) RDAPClient {
	return &rdapClient{client: client, base: strings.TrimRight(base, "/")}
}

func (r *rdapClient) Expiry(ctx context.Context, domain string) (time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.base+"/domain/"+url.PathEscape(domain), nil)
	if err != nil {
		return time.Time{}, err
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := r.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return time.Time{}, fmt.Errorf("rdap: %s for %q", resp.Status, domain)
	}
	var body struct {
		Events []struct {
			Action string `json:"eventAction"`
			Date   string `json:"eventDate"`
		} `json:"events"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil {
		return time.Time{}, fmt.Errorf("rdap: decode: %w", err)
	}
	for _, ev := range body.Events {
		if strings.EqualFold(ev.Action, "expiration") {
			t, err := time.Parse(time.RFC3339, ev.Date)
			if err != nil {
				return time.Time{}, fmt.Errorf("rdap: bad expiration date %q: %w", ev.Date, err)
			}
			return t.UTC(), nil
		}
	}
	return time.Time{}, errors.New("rdap: no expiration event in response")
}
