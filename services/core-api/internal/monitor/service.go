package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

const source = "core-api"

// Service holds the monitor business rules and publishes domain events.
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

// NotificationInput is one channel supplied with a create or update.
type NotificationInput struct {
	Type    model.NotificationType
	Enabled *bool
	Config  json.RawMessage
}

// CreateInput is what Create needs. Pointer fields take a default when nil.
type CreateInput struct {
	Type            model.MonitorType
	Name            string
	URL             string
	IntervalSeconds *int
	Retries         *int
	TimeoutSeconds  *int
	OwnerID         *uuid.UUID

	CertExpiryNotification   *bool
	DomainExpiryNotification *bool
	MaxRedirects             *int
	SaveErrorResponse        *bool
	AutoCreateIncident       *bool

	Method       model.HTTPMethod
	BodyEncoding model.BodyEncoding
	Body         string
	Headers      json.RawMessage

	AuthMethod    model.AuthMethod
	BasicUsername string
	BasicPassword string
	BearerToken   string

	Notifications []NotificationInput
}

// Create validates and stores a new monitor, then publishes monitor.created.
func (s *Service) Create(ctx context.Context, in CreateInput) (*model.Monitor, error) {
	now := s.now()
	m := &model.Monitor{
		Type:            in.Type,
		Name:            in.Name,
		URL:             in.URL,
		IntervalSeconds: valueOr(in.IntervalSeconds, model.MonitorDefaultInterval),
		Retries:         valueOr(in.Retries, 0),
		TimeoutSeconds:  valueOr(in.TimeoutSeconds, model.MonitorDefaultTimeout),
		Active:          true,
		OwnerID:         in.OwnerID,

		CertExpiryNotification:   valueOr(in.CertExpiryNotification, false),
		DomainExpiryNotification: valueOr(in.DomainExpiryNotification, false),
		MaxRedirects:             valueOr(in.MaxRedirects, model.MonitorDefaultMaxRedirects),
		SaveErrorResponse:        valueOr(in.SaveErrorResponse, false),
		AutoCreateIncident:       valueOr(in.AutoCreateIncident, false),

		Method:       in.Method,
		BodyEncoding: in.BodyEncoding,
		Body:         in.Body,
		Headers:      datatypes.JSON(in.Headers),

		AuthMethod:    in.AuthMethod,
		BasicUsername: in.BasicUsername,
		BasicPassword: in.BasicPassword,
		BearerToken:   in.BearerToken,

		LastStatus:  model.MonitorPending,
		NextCheckAt: &now,
	}
	if m.Type == "" {
		m.Type = model.MonitorHTTP
	}
	if m.Method == "" {
		m.Method = model.MethodGET
	}
	if m.BodyEncoding == "" {
		m.BodyEncoding = model.BodyJSON
	}
	if m.AuthMethod == "" {
		m.AuthMethod = model.AuthNone
	}

	notifs, err := buildNotifications(in.Notifications)
	if err != nil {
		return nil, err
	}
	m.Notifications = notifs

	if err := normalise(m); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectMonitorCreated, m)
	return m, nil
}

// Get loads one monitor with its notifications.
func (s *Service) Get(ctx context.Context, id uuid.UUID) (*model.Monitor, error) {
	return s.repo.Get(ctx, id)
}

// List pages through monitors.
func (s *Service) List(ctx context.Context, f ListFilter) ([]model.Monitor, int64, error) {
	if f.Status != "" && !f.Status.Valid() {
		return nil, 0, fmt.Errorf("%w: unknown status", ErrValidation)
	}
	return s.repo.List(ctx, f)
}

// ListHeartbeats pages through the probe history of one monitor, newest first.
func (s *Service) ListHeartbeats(ctx context.Context, id uuid.UUID, f HeartbeatFilter) ([]model.MonitorHeartbeat, int64, error) {
	if _, err := s.repo.Get(ctx, id); err != nil {
		return nil, 0, err
	}
	return s.repo.ListHeartbeats(ctx, id, f)
}

// UpdateInput carries optional field changes. A nil pointer leaves the field
// untouched. Notifications, when non-nil, replaces the whole set. An empty
// BasicPassword or BearerToken clears the stored secret.
type UpdateInput struct {
	Type            *model.MonitorType
	Name            *string
	URL             *string
	IntervalSeconds *int
	Retries         *int
	TimeoutSeconds  *int

	CertExpiryNotification   *bool
	DomainExpiryNotification *bool
	MaxRedirects             *int
	SaveErrorResponse        *bool
	AutoCreateIncident       *bool

	Method       *model.HTTPMethod
	BodyEncoding *model.BodyEncoding
	Body         *string
	Headers      json.RawMessage

	AuthMethod    *model.AuthMethod
	BasicUsername *string
	BasicPassword *string
	BearerToken   *string

	Notifications *[]NotificationInput
}

// Update applies field changes and reschedules the probe when the request shape changed.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (*model.Monitor, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	before := probeSignature(m)

	setIf(&m.Type, in.Type)
	setIf(&m.Name, in.Name)
	setIf(&m.URL, in.URL)
	setIf(&m.IntervalSeconds, in.IntervalSeconds)
	setIf(&m.Retries, in.Retries)
	setIf(&m.TimeoutSeconds, in.TimeoutSeconds)
	setIf(&m.CertExpiryNotification, in.CertExpiryNotification)
	setIf(&m.DomainExpiryNotification, in.DomainExpiryNotification)
	setIf(&m.MaxRedirects, in.MaxRedirects)
	setIf(&m.SaveErrorResponse, in.SaveErrorResponse)
	setIf(&m.AutoCreateIncident, in.AutoCreateIncident)
	setIf(&m.Method, in.Method)
	setIf(&m.BodyEncoding, in.BodyEncoding)
	setIf(&m.Body, in.Body)
	if in.Headers != nil {
		m.Headers = datatypes.JSON(in.Headers)
	}

	if in.AuthMethod != nil && *in.AuthMethod != m.AuthMethod {
		// Switching methods drops the credentials of the previous one.
		m.AuthMethod = *in.AuthMethod
		switch m.AuthMethod {
		case model.AuthBasic:
			m.BearerToken = ""
		case model.AuthBearer:
			m.BasicUsername, m.BasicPassword = "", ""
		default:
			m.BasicUsername, m.BasicPassword, m.BearerToken = "", "", ""
		}
	}
	setIf(&m.BasicUsername, in.BasicUsername)
	setIf(&m.BasicPassword, in.BasicPassword)
	setIf(&m.BearerToken, in.BearerToken)

	if in.Notifications != nil {
		notifs, err := buildNotifications(*in.Notifications)
		if err != nil {
			return nil, err
		}
		m.Notifications = notifs
	}

	if err := normalise(m); err != nil {
		return nil, err
	}
	if probeSignature(m) != before && m.Active {
		now := s.now()
		m.NextCheckAt = &now
		m.FailureCount = 0
	}

	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectMonitorUpdated, m)
	return m, nil
}

// Pause stops probing. Pausing a paused monitor is a no-op.
func (s *Service) Pause(ctx context.Context, id uuid.UUID) (*model.Monitor, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !m.Active {
		return m, nil
	}
	m.Active = false
	m.NextCheckAt = nil
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectMonitorUpdated, m)
	return m, nil
}

// Resume schedules the next probe immediately. Resuming an active monitor is a no-op.
func (s *Service) Resume(ctx context.Context, id uuid.UUID) (*model.Monitor, error) {
	m, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if m.Active {
		return m, nil
	}
	now := s.now()
	m.Active = true
	m.NextCheckAt = &now
	m.FailureCount = 0
	if err := s.repo.Update(ctx, m); err != nil {
		return nil, err
	}
	s.publish(ctx, messaging.SubjectMonitorUpdated, m)
	return m, nil
}

// Delete soft-deletes a monitor and publishes monitor.deleted.
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.publish(ctx, messaging.SubjectMonitorDeleted, map[string]uuid.UUID{"id": id})
	return nil
}

// normalise trims, applies defaults that depend on other fields and enforces
// every validation rule. It mutates m in place.
func normalise(m *model.Monitor) error {
	if !m.Type.Valid() {
		return fmt.Errorf("%w: type must be %s", ErrValidation, model.MonitorHTTP)
	}
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	m.URL = strings.TrimSpace(m.URL)
	if m.URL == "" {
		return fmt.Errorf("%w: url is required", ErrValidation)
	}
	u, err := url.Parse(m.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("%w: url must be an http or https URL", ErrValidation)
	}

	if m.IntervalSeconds < model.MonitorMinInterval {
		return fmt.Errorf("%w: interval_seconds must be at least %d", ErrValidation, model.MonitorMinInterval)
	}
	if m.Retries < 0 {
		return fmt.Errorf("%w: retries cannot be negative", ErrValidation)
	}
	if m.TimeoutSeconds <= 0 {
		return fmt.Errorf("%w: timeout_seconds must be positive", ErrValidation)
	}
	if m.TimeoutSeconds >= m.IntervalSeconds {
		return fmt.Errorf("%w: timeout_seconds must be less than interval_seconds", ErrValidation)
	}
	if m.MaxRedirects < 0 {
		return fmt.Errorf("%w: max_redirects cannot be negative", ErrValidation)
	}

	if !m.Method.Valid() {
		return fmt.Errorf("%w: method must be one of GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS", ErrValidation)
	}
	if !m.BodyEncoding.Valid() {
		return fmt.Errorf("%w: body_encoding must be one of json, form, xml", ErrValidation)
	}
	headers, err := normaliseHeaders(m.Headers)
	if err != nil {
		return err
	}
	m.Headers = headers

	if !m.AuthMethod.Valid() {
		return fmt.Errorf("%w: auth_method must be one of none, basic, bearer", ErrValidation)
	}
	m.BasicUsername = strings.TrimSpace(m.BasicUsername)
	m.BearerToken = strings.TrimSpace(m.BearerToken)
	switch m.AuthMethod {
	case model.AuthBasic:
		if m.BasicUsername == "" {
			return fmt.Errorf("%w: basic_username is required for basic auth", ErrValidation)
		}
		m.BearerToken = ""
	case model.AuthBearer:
		if m.BearerToken == "" {
			return fmt.Errorf("%w: bearer_token is required for bearer auth", ErrValidation)
		}
		m.BasicUsername, m.BasicPassword = "", ""
	default:
		m.BasicUsername, m.BasicPassword, m.BearerToken = "", "", ""
	}
	return nil
}

// normaliseHeaders checks the column is a JSON object of strings and returns
// it re-encoded. Empty input becomes {}.
func normaliseHeaders(raw datatypes.JSON) (datatypes.JSON, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
		return datatypes.JSON(`{}`), nil
	}
	var h map[string]string
	if err := json.Unmarshal(raw, &h); err != nil {
		return nil, fmt.Errorf("%w: headers must be a JSON object of string values", ErrValidation)
	}
	for k := range h {
		if strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("%w: header names cannot be empty", ErrValidation)
		}
	}
	out, err := json.Marshal(h)
	if err != nil {
		return nil, fmt.Errorf("%w: headers: %v", ErrValidation, err)
	}
	return datatypes.JSON(out), nil
}

// buildNotifications validates each channel and stores the config with defaults applied.
func buildNotifications(in []NotificationInput) ([]model.MonitorNotification, error) {
	out := make([]model.MonitorNotification, 0, len(in))
	for i, n := range in {
		if !n.Type.Valid() {
			return nil, fmt.Errorf("%w: notifications[%d].type must be one of discord, webpush, webhook", ErrValidation, i)
		}
		cfg, err := model.DecodeNotificationConfig(n.Type, n.Config)
		if err != nil {
			return nil, fmt.Errorf("%w: notifications[%d]: %v", ErrValidation, i, err)
		}
		if err := cfg.Validate(); err != nil {
			return nil, fmt.Errorf("%w: notifications[%d]: %v", ErrValidation, i, err)
		}
		raw, err := json.Marshal(cfg)
		if err != nil {
			return nil, fmt.Errorf("%w: notifications[%d]: %v", ErrValidation, i, err)
		}
		out = append(out, model.MonitorNotification{
			Type:    n.Type,
			Enabled: valueOr(n.Enabled, true),
			Config:  datatypes.JSON(raw),
		})
	}
	return out, nil
}

// probeSignature captures every field that changes what the scheduler sends,
// so Update can tell whether the next probe should run immediately.
func probeSignature(m *model.Monitor) string {
	return strings.Join([]string{
		string(m.Type), m.URL, string(m.Method), string(m.BodyEncoding), m.Body, string(m.Headers),
		string(m.AuthMethod), m.BasicUsername, m.BasicPassword, m.BearerToken,
		fmt.Sprint(m.TimeoutSeconds), fmt.Sprint(m.IntervalSeconds), fmt.Sprint(m.MaxRedirects),
	}, "\x00")
}

func valueOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}

func setIf[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
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
