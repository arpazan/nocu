package uptime

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

func TestShouldWarn(t *testing.T) {
	th := []int{7, 14, 30}
	cases := map[int]bool{30: true, 29: false, 14: true, 10: false, 7: true, 6: true, 0: true, -3: true, 31: false}
	for days, want := range cases {
		if got := ShouldWarn(days, th); got != want {
			t.Errorf("days=%d got %v want %v", days, got, want)
		}
	}
	if ShouldWarn(1, nil) {
		t.Error("no thresholds should never warn")
	}
}

func TestDaysLeft(t *testing.T) {
	now := time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)
	if d := DaysLeft(now, now.Add(7*24*time.Hour+time.Hour)); d != 7 {
		t.Fatalf("got %d", d)
	}
	if d := DaysLeft(now, now.Add(-2*24*time.Hour)); d != -2 {
		t.Fatalf("got %d", d)
	}
}

func TestRegistrableDomain(t *testing.T) {
	cases := map[string]string{
		"www.example.com":        "example.com",
		"example.com":            "example.com",
		"a.b.c.example.org":      "example.org",
		"api.example.co.th":      "example.co.th",
		"example.co.th":          "example.co.th",
		"shop.example.co.uk":     "example.co.uk",
		"Example.COM.":           "example.com",
		"localhost":              "localhost",
		"deep.sub.example.ac.th": "example.ac.th",
	}
	for in, want := range cases {
		if got := RegistrableDomain(in); got != want {
			t.Errorf("%s: got %s want %s", in, got, want)
		}
	}
}

type fakeRDAP struct {
	expiry time.Time
	err    error
	asked  []string
}

func (f *fakeRDAP) Expiry(_ context.Context, domain string) (time.Time, error) {
	f.asked = append(f.asked, domain)
	return f.expiry, f.err
}

func TestExpiryCheckerWarnsAndStores(t *testing.T) {
	now := time.Date(2026, 11, 5, 6, 0, 0, 0, time.UTC)
	certExp := now.Add(14*24*time.Hour + time.Hour) // exactly 14 days -> warn
	domainExp := now.Add(100 * 24 * time.Hour)      // far away -> no warn

	m := model.Monitor{
		Base: model.Base{ID: uuid.New()}, Name: "site", URL: "https://www.example.co.th/health", Active: true,
		CertExpiryNotification: true, DomainExpiryNotification: true, CertExpiresAt: &certExp, LastStatus: model.MonitorUp,
		Notifications: []model.MonitorNotification{
			{Base: model.Base{ID: uuid.New()}, Type: model.NotificationWebhook, Enabled: true, Config: []byte(`{"post_url":"https://hooks.example/x"}`)},
		},
	}
	noNotify := model.Monitor{Base: model.Base{ID: uuid.New()}, Name: "plain", URL: "https://plain.example", Active: true}
	store := newFakeStore(m, noNotify)
	broker := messaging.NewMemory()
	rec := &recordingSender{}
	disp := NewDispatcherWithSenders(slog.Default(), map[model.NotificationType]Sender{model.NotificationWebhook: rec})
	rdap := &fakeRDAP{expiry: domainExp}
	c := NewExpiryChecker(store, broker, disp, slog.Default(), ExpiryOptions{WarnDays: []int{30, 7, 14}, RDAP: rdap})
	c.now = func() time.Time { return now }

	if err := c.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(rdap.asked) != 1 || rdap.asked[0] != "example.co.th" {
		t.Fatalf("rdap asked %v", rdap.asked)
	}
	if store.monitors[0].DomainExpiresAt == nil || !store.monitors[0].DomainExpiresAt.Equal(domainExp) {
		t.Fatal("domain expiry not stored")
	}
	if subs := broker.Subjects(); len(subs) != 1 || subs[0] != messaging.SubjectMonitorCertExpiring {
		t.Fatalf("published %v", subs)
	}
	if len(rec.events) != 1 || rec.events[0].Kind != KindCertExpiring || rec.events[0].DaysLeft != 14 {
		t.Fatalf("notified %+v", rec.events)
	}

	// Domain now inside the window and RDAP failing for cert-only lookups is tolerated.
	rdap.expiry = now.Add(3 * 24 * time.Hour)
	broker.Published = nil
	if err := c.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	subs := broker.Subjects()
	if len(subs) != 2 || subs[1] != messaging.SubjectMonitorDomainExpiring {
		t.Fatalf("published %v", subs)
	}

	rdap.err = errors.New("rdap down")
	if err := c.Run(context.Background()); err != nil {
		t.Fatalf("lookup failures must not fail the job: %v", err)
	}
}

func TestRDAPClientParsesExpiration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/domain/example.com" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write([]byte(`{"events":[{"eventAction":"registration","eventDate":"2000-01-01T00:00:00Z"},{"eventAction":"expiration","eventDate":"2027-03-04T05:06:07Z"}]}`))
	}))
	defer srv.Close()
	c := NewRDAPClient(srv.Client(), srv.URL)
	got, err := c.Expiry(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(time.Date(2027, 3, 4, 5, 6, 7, 0, time.UTC)) {
		t.Fatalf("got %v", got)
	}
	if _, err := c.Expiry(context.Background(), "missing.com"); err == nil {
		t.Fatal("404 should error")
	}
}
