package incident

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

const source = "core-api"

// Service holds the incident business rules and publishes domain events.
type Service struct {
	repo Repository
	pub  messaging.Publisher
	log  *slog.Logger
	now  func() time.Time
}

// NewService wires the service.
func NewService(repo Repository, pub messaging.Publisher, log *slog.Logger) *Service {
	return &Service{repo: repo, pub: pub, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// CreateInput is what Create needs.
type CreateInput struct {
	Title       string
	Description string
	Severity    model.Severity
	Source      string
	ReporterID  *uuid.UUID
}

// Create validates and stores a new incident, then publishes incident.created.
func (s *Service) Create(ctx context.Context, in CreateInput) (*model.Incident, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if !in.Severity.Valid() {
		return nil, fmt.Errorf("%w: severity must be one of critical, high, medium, low", ErrValidation)
	}

	inc := &model.Incident{
		Title:       title,
		Description: strings.TrimSpace(in.Description),
		Severity:    in.Severity,
		Status:      model.IncidentOpen,
		Source:      strings.TrimSpace(in.Source),
		ReporterID:  in.ReporterID,
	}
	if err := s.repo.Create(ctx, inc); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectIncidentCreated, inc)
	return inc, nil
}

// Get loads one incident.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*model.Incident, error) {
	return s.repo.Get(ctx, id)
}

// List pages through incidents.
func (s *Service) List(ctx context.Context, f ListFilter) ([]model.Incident, int64, error) {
	if f.Status != "" && !f.Status.Valid() {
		return nil, 0, fmt.Errorf("%w: unknown status", ErrValidation)
	}
	if f.Severity != "" && !f.Severity.Valid() {
		return nil, 0, fmt.Errorf("%w: unknown severity", ErrValidation)
	}
	return s.repo.List(ctx, f)
}

// UpdateInput carries optional field changes.
type UpdateInput struct {
	Title       *string
	Description *string
	Severity    *model.Severity
	Status      *model.IncidentStatus
}

// Update applies field changes and enforces the status state machine.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*model.Incident, error) {
	inc, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if in.Title != nil {
		t := strings.TrimSpace(*in.Title)
		if t == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", ErrValidation)
		}
		inc.Title = t
	}
	if in.Description != nil {
		inc.Description = strings.TrimSpace(*in.Description)
	}
	if in.Severity != nil {
		if !in.Severity.Valid() {
			return nil, fmt.Errorf("%w: unknown severity", ErrValidation)
		}
		inc.Severity = *in.Severity
	}
	if in.Status != nil && *in.Status != inc.Status {
		if !in.Status.Valid() {
			return nil, fmt.Errorf("%w: unknown status", ErrValidation)
		}
		if !inc.Status.CanTransitionTo(*in.Status) {
			return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, inc.Status, *in.Status)
		}
		s.applyStatus(inc, *in.Status)
	}

	if err := s.repo.Update(ctx, inc); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectIncidentUpdated, inc)
	return inc, nil
}

func (s *Service) applyStatus(inc *model.Incident, next model.IncidentStatus) {
	now := s.now()
	inc.Status = next
	switch next {
	case model.IncidentAcknowledged:
		inc.AcknowledgedAt = &now
	case model.IncidentResolved:
		inc.ResolvedAt = &now
	case model.IncidentClosed:
		inc.ClosedAt = &now
	case model.IncidentOpen:
		inc.ResolvedAt = nil
		inc.ClosedAt = nil
	}
}

// Delete soft-deletes an incident and publishes incident.deleted.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, messaging.SubjectIncidentDeleted, map[string]uuid.UUID{"id": id})
	return nil
}

// publish is best-effort: the write already succeeded, so a broker failure is logged, not returned.
func (s *Service) publish(ctx context.Context, subject string, v any) {
	if s.pub == nil {
		return
	}
	if err := messaging.PublishJSON(ctx, s.pub, subject, source, v); err != nil {
		s.log.ErrorContext(ctx, "publish failed", "subject", subject, "err", err)
	}
}
