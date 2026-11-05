package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/Pzquzer/Nocu/shared/messaging"
)

func TestRegisterValidatesJobs(t *testing.T) {
	r := NewRunner(nil, nil, slog.Default(), time.Second)

	if err := r.Register(Job{Name: "", Spec: "* * * * *", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected error for empty name")
	}
	if err := r.Register(Job{Name: "x", Spec: "not a spec", Run: func(context.Context) error { return nil }}); err == nil {
		t.Fatal("expected error for bad spec")
	}
	if err := r.Register(Job{Name: "x", Spec: "@every 1m"}); err == nil {
		t.Fatal("expected error for nil Run")
	}
	ok := Job{Name: "x", Spec: "@every 1m", Run: func(context.Context) error { return nil }}
	if err := r.Register(ok); err != nil {
		t.Fatalf("valid job rejected: %v", err)
	}
	if err := r.Register(ok); err == nil {
		t.Fatal("expected duplicate error")
	}
	if len(r.Jobs()) != 1 {
		t.Fatalf("jobs = %d", len(r.Jobs()))
	}
}

func TestDefaultJobsHaveValidSpecs(t *testing.T) {
	for _, j := range DefaultJobs(JobDeps{Log: slog.Default()}) {
		if err := j.Validate(); err != nil {
			t.Errorf("%s: %v", j.Name, err)
		}
	}
}

func TestLockKeyIsStableAndDistinct(t *testing.T) {
	if lockKey("a") != lockKey("a") {
		t.Fatal("lock key not stable")
	}
	if lockKey("a") == lockKey("b") {
		t.Fatal("lock keys collide")
	}
}

func TestRunGuardedPublishesOutcome(t *testing.T) {
	broker := messaging.NewMemory()
	r := NewRunner(nil, broker, slog.Default(), time.Second)

	okJob := Job{Name: "ok", Spec: "@every 1m", Run: func(context.Context) error { return nil }}
	failJob := Job{Name: "fail", Spec: "@every 1m", Run: func(context.Context) error { return errors.New("boom") }}
	panicJob := Job{Name: "panic", Spec: "@every 1m", Run: func(context.Context) error { panic("eek") }}

	r.execute(context.Background(), okJob)
	r.execute(context.Background(), failJob)
	r.execute(context.Background(), panicJob)

	want := []string{messaging.SubjectJobCompleted, messaging.SubjectJobFailed, messaging.SubjectJobFailed}
	got := broker.Subjects()
	if len(got) != len(want) {
		t.Fatalf("published %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("published %v want %v", got, want)
		}
	}
}
