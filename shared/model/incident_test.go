package model

import "testing"

func TestIncidentStatusTransitions(t *testing.T) {
	cases := []struct {
		from, to IncidentStatus
		want     bool
	}{
		{IncidentOpen, IncidentAcknowledged, true},
		{IncidentOpen, IncidentClosed, true},
		{IncidentAcknowledged, IncidentOpen, false},
		{IncidentAcknowledged, IncidentResolved, true},
		{IncidentResolved, IncidentOpen, true},
		{IncidentClosed, IncidentOpen, false},
		{IncidentOpen, IncidentOpen, false},
	}
	for _, c := range cases {
		if got := c.from.CanTransitionTo(c.to); got != c.want {
			t.Errorf("%s -> %s: got %v want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestSeverityValid(t *testing.T) {
	if !SeverityCritical.Valid() || Severity("nope").Valid() {
		t.Fatal("severity validation broken")
	}
}
