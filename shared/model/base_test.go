package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBaseBeforeCreate(t *testing.T) {
	t.Run("assigns id when unset", func(t *testing.T) {
		var b Base
		if err := b.BeforeCreate(nil); err != nil {
			t.Fatal(err)
		}
		if b.ID == uuid.Nil {
			t.Error("expected a generated UUID")
		}
	})

	t.Run("keeps caller-provided id", func(t *testing.T) {
		want := uuid.New()
		b := Base{ID: want}
		if err := b.BeforeCreate(nil); err != nil {
			t.Fatal(err)
		}
		if b.ID != want {
			t.Errorf("id = %s; want %s", b.ID, want)
		}
	})
}

func TestAllListsEveryModelOnce(t *testing.T) {
	want := []string{"Role", "User", "Incident", "Event", "ScheduledJob"}

	all := All()
	if len(all) != len(want) {
		t.Fatalf("All() has %d entries; want %d (%v)", len(all), len(want), want)
	}
	seen := map[string]bool{}
	for _, m := range all {
		key := typeName(m)
		if seen[key] {
			t.Errorf("%s listed twice", key)
		}
		seen[key] = true
	}
	for _, name := range want {
		if !seen[name] {
			t.Errorf("All() is missing %s", name)
		}
	}
	// Role and User must precede anything referencing them.
	if typeName(all[0]) != "Role" || typeName(all[1]) != "User" {
		t.Errorf("All() must list Role then User first for FK ordering; got %s, %s", typeName(all[0]), typeName(all[1]))
	}
}

func typeName(v any) string {
	switch v.(type) {
	case *Role:
		return "Role"
	case *User:
		return "User"
	case *Incident:
		return "Incident"
	case *Event:
		return "Event"
	case *ScheduledJob:
		return "ScheduledJob"
	default:
		return fmt.Sprintf("unknown(%T)", v)
	}
}

func TestIncidentStatusValid(t *testing.T) {
	tests := []struct {
		status IncidentStatus
		want   bool
	}{
		{IncidentOpen, true},
		{IncidentAcknowledged, true},
		{IncidentResolved, true},
		{IncidentClosed, true},
		{"", false},
		{"OPEN", false},
		{"pending", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.Valid(); got != tt.want {
				t.Errorf("%q.Valid() = %v; want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestBodyEncodingContentType(t *testing.T) {
	tests := []struct {
		enc  BodyEncoding
		want string
	}{
		{BodyJSON, "application/json"},
		{BodyForm, "application/x-www-form-urlencoded"},
		{BodyXML, "application/xml"},
		{"", "application/json"},
		{"unknown", "application/json"},
	}
	for _, tt := range tests {
		t.Run(string(tt.enc), func(t *testing.T) {
			if got := tt.enc.ContentType(); got != tt.want {
				t.Errorf("%q.ContentType() = %q; want %q", tt.enc, got, tt.want)
			}
		})
	}
}

func TestMonitorDurations(t *testing.T) {
	m := &Monitor{IntervalSeconds: 90, TimeoutSeconds: 15}
	if got := m.Interval(); got != 90*time.Second {
		t.Errorf("Interval() = %v; want 90s", got)
	}
	if got := m.Timeout(); got != 15*time.Second {
		t.Errorf("Timeout() = %v; want 15s", got)
	}
	zero := &Monitor{}
	if zero.Interval() != 0 || zero.Timeout() != 0 {
		t.Errorf("zero monitor durations = %v / %v; want 0", zero.Interval(), zero.Timeout())
	}
}

func TestUserRoleNames(t *testing.T) {
	tests := []struct {
		name  string
		roles []Role
		want  []string
	}{
		{"no roles", nil, []string{}},
		{"single", []Role{{Name: RoleAdmin}}, []string{RoleAdmin}},
		{"preserves order", []Role{{Name: RoleViewer}, {Name: RoleOperator}}, []string{RoleViewer, RoleOperator}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (&User{Roles: tt.roles}).RoleNames()
			if got == nil {
				t.Fatal("RoleNames must return a non-nil slice so it marshals as []")
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v; want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q; want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMonitorNotificationDecode(t *testing.T) {
	tests := []struct {
		name    string
		n       MonitorNotification
		wantErr bool
	}{
		{
			name: "valid webhook",
			n:    MonitorNotification{Type: NotificationWebhook, Config: []byte(`{"post_url":"https://hook.example"}`)},
		},
		{
			name:    "unknown type",
			n:       MonitorNotification{Type: "pager", Config: []byte(`{}`)},
			wantErr: true,
		},
		{
			name:    "empty config",
			n:       MonitorNotification{Type: NotificationWebhook},
			wantErr: true,
		},
		{
			name:    "unknown field rejected",
			n:       MonitorNotification{Type: NotificationWebhook, Config: []byte(`{"url":"x"}`)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := tt.n.Decode()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got config %+v", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := cfg.(*WebhookConfig); !ok {
				t.Errorf("config type = %T; want *WebhookConfig", cfg)
			}
		})
	}
}
