// Package monitor implements the uptime monitor CRUD exposed by core-api.
package monitor

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// Sentinel errors.
var (
	ErrNotFound   = errors.New("monitor: not found")
	ErrValidation = errors.New("monitor: validation failed")
)

// updateOmitted lists the columns Update never writes: identity columns,
// the association, and probe state maintained by the scheduler service.
var updateOmitted = []string{"id", "created_at", "Notifications", "last_status", "last_checked_at", "cert_expires_at", "domain_expires_at"}

// ListFilter narrows List.
type ListFilter struct {
	Active *bool
	Status model.MonitorStatus
	Limit  int
	Offset int
}

func (f *ListFilter) normalise() {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// HeartbeatFilter narrows ListHeartbeats.
type HeartbeatFilter struct {
	Important bool
	Limit     int
	Offset    int
}

func (f *HeartbeatFilter) normalise() {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
}

// Repository is the persistence boundary for monitors.
//
// Get and List preload Notifications. Update saves the monitor columns and
// replaces the notification set atomically.
type Repository interface {
	Create(ctx context.Context, m *model.Monitor) error
	Get(ctx context.Context, id uuid.UUID) (*model.Monitor, error)
	List(ctx context.Context, f ListFilter) ([]model.Monitor, int64, error)
	Update(ctx context.Context, m *model.Monitor) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListHeartbeats(ctx context.Context, monitorID uuid.UUID, f HeartbeatFilter) ([]model.MonitorHeartbeat, int64, error)
}

type gormRepository struct{ db *gorm.DB }

// NewGormRepository returns a Repository backed by GORM.
func NewGormRepository(db *gorm.DB) Repository { return &gormRepository{db: db} }

func (r *gormRepository) Create(ctx context.Context, m *model.Monitor) error {
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("monitor: create: %w", err)
	}
	return nil
}

func (r *gormRepository) Get(ctx context.Context, id uuid.UUID) (*model.Monitor, error) {
	var m model.Monitor
	err := r.db.WithContext(ctx).Preload("Notifications").First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("monitor: get: %w", err)
	}
	return &m, nil
}

func (r *gormRepository) List(ctx context.Context, f ListFilter) ([]model.Monitor, int64, error) {
	f.normalise()
	q := r.db.WithContext(ctx).Model(&model.Monitor{})
	if f.Active != nil {
		q = q.Where("active = ?", *f.Active)
	}
	if f.Status != "" {
		q = q.Where("last_status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("monitor: count: %w", err)
	}
	var items []model.Monitor
	err := q.Preload("Notifications").Order("created_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("monitor: list: %w", err)
	}
	return items, total, nil
}

// Update writes the configuration columns and replaces the notification rows
// in one transaction. Columns the scheduler alone maintains (last_status,
// last_checked_at, cert_expires_at, domain_expires_at) are never written here
// so a probe finishing between load and save is not clobbered. next_check_at
// and failure_count are written because the service sets them deliberately on
// reschedule, pause and resume.
func (r *gormRepository) Update(ctx context.Context, m *model.Monitor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(m).Select("*").Omit(updateOmitted...).Updates(m)
		if res.Error != nil {
			return fmt.Errorf("monitor: update: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		if err := tx.Unscoped().Where("monitor_id = ?", m.ID).Delete(&model.MonitorNotification{}).Error; err != nil {
			return fmt.Errorf("monitor: clear notifications: %w", err)
		}
		if len(m.Notifications) == 0 {
			return nil
		}
		for i := range m.Notifications {
			m.Notifications[i].MonitorID = m.ID
			m.Notifications[i].ID = uuid.Nil
		}
		if err := tx.Create(&m.Notifications).Error; err != nil {
			return fmt.Errorf("monitor: save notifications: %w", err)
		}
		return nil
	})
}

func (r *gormRepository) Delete(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&model.Monitor{}, "id = ?", id)
	if res.Error != nil {
		return fmt.Errorf("monitor: delete: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *gormRepository) ListHeartbeats(ctx context.Context, monitorID uuid.UUID, f HeartbeatFilter) ([]model.MonitorHeartbeat, int64, error) {
	f.normalise()
	q := r.db.WithContext(ctx).Model(&model.MonitorHeartbeat{}).Where("monitor_id = ?", monitorID)
	if f.Important {
		q = q.Where("important = ?", true)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("monitor: count heartbeats: %w", err)
	}
	var items []model.MonitorHeartbeat
	if err := q.Order("checked_at DESC").Limit(f.Limit).Offset(f.Offset).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("monitor: list heartbeats: %w", err)
	}
	return items, total, nil
}
