package uptime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Store is the persistence boundary of the worker and the expiry checker.
type Store interface {
	// Due claims up to limit active monitors whose next check is at or before
	// now, pushes their next_check_at forward by one interval so other replicas
	// skip them, and returns them with notifications loaded.
	Due(ctx context.Context, now time.Time, limit int) ([]model.Monitor, error)
	// ExpiryCandidates returns active monitors with any expiry notification enabled.
	ExpiryCandidates(ctx context.Context) ([]model.Monitor, error)
	// SaveHeartbeat inserts one probe result.
	SaveHeartbeat(ctx context.Context, hb *model.MonitorHeartbeat) error
	// UpdateMonitor sets probe-state columns on one monitor without touching updated_at.
	UpdateMonitor(ctx context.Context, id uuid.UUID, fields map[string]any) error
	// CreateIncidentIfNone inserts inc unless its monitor already has an open or
	// acknowledged incident. It reports whether a row was inserted.
	CreateIncidentIfNone(ctx context.Context, inc *model.Incident) (bool, error)
}

// activeIncidentStatuses are the incident states that block a new auto-created incident.
var activeIncidentStatuses = []model.IncidentStatus{model.IncidentOpen, model.IncidentAcknowledged}

type gormStore struct{ db *gorm.DB }

// NewGormStore returns a Store backed by GORM.
func NewGormStore(db *gorm.DB) Store { return &gormStore{db: db} }

func (s *gormStore) Due(ctx context.Context, now time.Time, limit int) ([]model.Monitor, error) {
	if limit <= 0 {
		limit = 10
	}
	var due []model.Monitor
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("active = ? AND (next_check_at IS NULL OR next_check_at <= ?)", true, now).
			Order("next_check_at NULLS FIRST").
			Limit(limit).
			Find(&due).Error
		if err != nil {
			return fmt.Errorf("select due: %w", err)
		}
		if len(due) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, 0, len(due))
		for i := range due {
			interval := due[i].Interval()
			if interval <= 0 {
				interval = time.Duration(model.MonitorDefaultInterval) * time.Second
			}
			next := now.Add(interval)
			if err := tx.Model(&model.Monitor{}).Where("id = ?", due[i].ID).
				UpdateColumn("next_check_at", next).Error; err != nil {
				return fmt.Errorf("claim %s: %w", due[i].ID, err)
			}
			due[i].NextCheckAt = &next
			ids = append(ids, due[i].ID)
		}
		var notes []model.MonitorNotification
		if err := tx.Where("monitor_id IN ?", ids).Find(&notes).Error; err != nil {
			return fmt.Errorf("load notifications: %w", err)
		}
		byMonitor := map[uuid.UUID][]model.MonitorNotification{}
		for _, n := range notes {
			byMonitor[n.MonitorID] = append(byMonitor[n.MonitorID], n)
		}
		for i := range due {
			due[i].Notifications = byMonitor[due[i].ID]
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("uptime: due: %w", err)
	}
	return due, nil
}

func (s *gormStore) ExpiryCandidates(ctx context.Context) ([]model.Monitor, error) {
	var out []model.Monitor
	err := s.db.WithContext(ctx).
		Preload("Notifications").
		Where("active = ? AND (cert_expiry_notification = ? OR domain_expiry_notification = ?)", true, true, true).
		Find(&out).Error
	if err != nil {
		return nil, fmt.Errorf("uptime: expiry candidates: %w", err)
	}
	return out, nil
}

func (s *gormStore) SaveHeartbeat(ctx context.Context, hb *model.MonitorHeartbeat) error {
	if err := s.db.WithContext(ctx).Create(hb).Error; err != nil {
		return fmt.Errorf("uptime: save heartbeat: %w", err)
	}
	return nil
}

func (s *gormStore) UpdateMonitor(ctx context.Context, id uuid.UUID, fields map[string]any) error {
	if len(fields) == 0 {
		return nil
	}
	err := s.db.WithContext(ctx).Model(&model.Monitor{}).Where("id = ?", id).UpdateColumns(fields).Error
	if err != nil {
		return fmt.Errorf("uptime: update monitor %s: %w", id, err)
	}
	return nil
}

func (s *gormStore) CreateIncidentIfNone(ctx context.Context, inc *model.Incident) (bool, error) {
	if inc.MonitorID == nil {
		return false, errors.New("uptime: create incident: monitor_id is required")
	}
	created := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var active int64
		err := tx.Model(&model.Incident{}).
			Where("monitor_id = ? AND status IN ?", *inc.MonitorID, activeIncidentStatuses).
			Count(&active).Error
		if err != nil {
			return fmt.Errorf("count active: %w", err)
		}
		if active > 0 {
			return nil
		}
		if err := tx.Create(inc).Error; err != nil {
			// Another replica won the race; the partial unique index rejected ours.
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil
			}
			return fmt.Errorf("insert: %w", err)
		}
		created = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("uptime: create incident for %s: %w", *inc.MonitorID, err)
	}
	return created, nil
}
