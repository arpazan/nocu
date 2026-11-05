package messaging

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSubjectMatches(t *testing.T) {
	cases := []struct {
		pattern, subject string
		want             bool
	}{
		{"incident.>", "incident.created", true},
		{"incident.>", "incident.sla.breached", true},
		{"incident.>", "incident", false},
		{"incident.*", "incident.created", true},
		{"incident.*", "incident.a.b", false},
		{"job.completed", "job.completed", true},
		{"job.completed", "job.failed", false},
	}
	for _, c := range cases {
		if got := subjectMatches(c.pattern, c.subject); got != c.want {
			t.Errorf("%s vs %s: got %v want %v", c.pattern, c.subject, got, c.want)
		}
	}
}

func TestMemoryPublishJSONFansOut(t *testing.T) {
	m := NewMemory()
	var got []Message
	sub, _ := m.Subscribe(context.Background(), SubjectIncidentAll, "", func(_ context.Context, msg Message) error {
		got = append(got, msg)
		return nil
	})
	defer sub.Unsubscribe()

	if err := PublishJSON(context.Background(), m, SubjectIncidentCreated, "test", map[string]string{"id": "1"}); err != nil {
		t.Fatal(err)
	}
	if err := PublishJSON(context.Background(), m, SubjectJobCompleted, "test", nil); err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].Subject != SubjectIncidentCreated {
		t.Fatalf("expected one incident message, got %+v", got)
	}
	var env Envelope
	if err := json.Unmarshal(got[0].Data, &env); err != nil {
		t.Fatal(err)
	}
	if env.Type != SubjectIncidentCreated || env.Source != "test" || string(env.Data) != `{"id":"1"}` {
		t.Fatalf("bad envelope: %+v", env)
	}
	if len(m.Subjects()) != 2 {
		t.Fatalf("expected 2 published, got %d", len(m.Subjects()))
	}
}
