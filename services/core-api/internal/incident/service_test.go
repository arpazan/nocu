package incident

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

type fakeRepo struct{ items map[uuid.UUID]*model.Incident }

func newFakeRepo() *fakeRepo { return &fakeRepo{items: map[uuid.UUID]*model.Incident{}} }

func (f *fakeRepo) Create(_ context.Context, in *model.Incident) error {
	in.ID = uuid.New()
	cp := *in
	f.items[in.ID] = &cp
	return nil
}

func (f *fakeRepo) Get(_ context.Context, id uuid.UUID) (*model.Incident, error) {
	if in, ok := f.items[id]; ok {
		cp := *in
		return &cp, nil
	}
	return nil, ErrNotFound
}

func (f *fakeRepo) List(_ context.Context, _ ListFilter) ([]model.Incident, int64, error) {
	out := make([]model.Incident, 0, len(f.items))
	for _, in := range f.items {
		out = append(out, *in)
	}
	return out, int64(len(out)), nil
}

func (f *fakeRepo) Update(_ context.Context, in *model.Incident) error {
	if _, ok := f.items[in.ID]; !ok {
		return ErrNotFound
	}
	cp := *in
	f.items[in.ID] = &cp
	return nil
}

func (f *fakeRepo) Delete(_ context.Context, id uuid.UUID) error {
	if _, ok := f.items[id]; !ok {
		return ErrNotFound
	}
	delete(f.items, id)
	return nil
}

func newService() (*Service, *messaging.Memory) {
	broker := messaging.NewMemory()
	return NewService(newFakeRepo(), broker, slog.Default()), broker
}

func ptr[T any](v T) *T { return &v }

func TestCreatePublishesEvent(t *testing.T) {
	svc, broker := newService()
	inc, err := svc.Create(context.Background(), CreateInput{Title: "  Router down ", Severity: model.SeverityCritical})
	if err != nil {
		t.Fatal(err)
	}
	if inc.Title != "Router down" || inc.Status != model.IncidentOpen {
		t.Fatalf("unexpected incident: %+v", inc)
	}
	if subs := broker.Subjects(); len(subs) != 1 || subs[0] != messaging.SubjectIncidentCreated {
		t.Fatalf("expected incident.created, got %v", subs)
	}
}

func TestCreateValidation(t *testing.T) {
	svc, _ := newService()
	if _, err := svc.Create(context.Background(), CreateInput{Title: "", Severity: model.SeverityLow}); !errors.Is(err, ErrValidation) {
		t.Fatalf("empty title: %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateInput{Title: "x", Severity: "bogus"}); !errors.Is(err, ErrValidation) {
		t.Fatalf("bad severity: %v", err)
	}
}

func TestUpdateStatusTransitions(t *testing.T) {
	svc, broker := newService()
	ctx := context.Background()
	inc, _ := svc.Create(ctx, CreateInput{Title: "x", Severity: model.SeverityHigh})

	ack, err := svc.Update(ctx, inc.ID, UpdateInput{Status: ptr(model.IncidentAcknowledged)})
	if err != nil {
		t.Fatal(err)
	}
	if ack.Status != model.IncidentAcknowledged || ack.AcknowledgedAt == nil {
		t.Fatalf("ack not applied: %+v", ack)
	}

	if _, err := svc.Update(ctx, inc.ID, UpdateInput{Status: ptr(model.IncidentOpen)}); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}

	closed, err := svc.Update(ctx, inc.ID, UpdateInput{Status: ptr(model.IncidentClosed), Title: ptr("renamed")})
	if err != nil {
		t.Fatal(err)
	}
	if closed.ClosedAt == nil || closed.Title != "renamed" {
		t.Fatalf("close not applied: %+v", closed)
	}

	want := []string{messaging.SubjectIncidentCreated, messaging.SubjectIncidentUpdated, messaging.SubjectIncidentUpdated}
	got := broker.Subjects()
	if len(got) != len(want) {
		t.Fatalf("published %v want %v", got, want)
	}
}

func TestDeleteNotFound(t *testing.T) {
	svc, _ := newService()
	if err := svc.Delete(context.Background(), uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
