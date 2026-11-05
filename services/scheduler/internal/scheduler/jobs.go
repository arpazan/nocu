package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

// JobDeps is what the built-in jobs need.
type JobDeps struct {
	DB             *gorm.DB
	Pub            messaging.Publisher
	Log            *slog.Logger
	SLAAckDeadline time.Duration
	EventRetention time.Duration
	// HeartbeatRetention is how long monitor heartbeats are kept.
	HeartbeatRetention time.Duration
	// ExpiryCheck runs the certificate and domain expiry sweep. Nil skips the job.
	ExpiryCheck func(context.Context) error
}

// DefaultJobs returns the jobs the scheduler ships with.
func DefaultJobs(d JobDeps) []Job {
	jobs := []Job{
		{Name: "heartbeat", Spec: "* * * * *", Run: heartbeat(d)},
		{Name: "incident_sla_check", Spec: "*/5 * * * *", Run: incidentSLACheck(d)},
		{Name: "event_cleanup", Spec: "0 3 * * *", Run: eventCleanup(d)},
		{Name: "heartbeat_cleanup", Spec: "30 3 * * *", Run: heartbeatCleanup(d)},
	}
	if d.ExpiryCheck != nil {
		jobs = append(jobs, Job{Name: "monitor_expiry_check", Spec: "0 6 * * *", Run: d.ExpiryCheck})
	}
	return jobs
}

// heartbeat publishes a liveness event once a minute so dashboards can show the scheduler is up.
func heartbeat(d JobDeps) func(context.Context) error {
	return func(ctx context.Context) error {
		return messaging.PublishJSON(ctx, d.Pub, messaging.SubjectJobHeartbeat, source, map[string]any{
			"at": time.Now().UTC(),
		})
	}
}

// incidentSLACheck flags critical incidents still unacknowledged past the deadline.
func incidentSLACheck(d JobDeps) func(context.Context) error {
	return func(ctx context.Context) error {
		cutoff := time.Now().UTC().Add(-d.SLAAckDeadline)
		var breached []model.Incident
		err := d.DB.WithContext(ctx).
			Where("status = ? AND severity = ? AND created_at < ?", model.IncidentOpen, model.SeverityCritical, cutoff).
			Find(&breached).Error
		if err != nil {
			return fmt.Errorf("query breached incidents: %w", err)
		}
		for i := range breached {
			inc := breached[i]
			if err := messaging.PublishJSON(ctx, d.Pub, messaging.SubjectIncidentSLABreached, source, map[string]any{
				"id":         inc.ID,
				"title":      inc.Title,
				"created_at": inc.CreatedAt,
				"deadline":   d.SLAAckDeadline.String(),
			}); err != nil {
				return err
			}
		}
		d.Log.Info("sla check complete", "breached", len(breached))
		return nil
	}
}

// eventCleanup hard-deletes consumer events older than the retention window.
func eventCleanup(d JobDeps) func(context.Context) error {
	return func(ctx context.Context) error {
		cutoff := time.Now().UTC().Add(-d.EventRetention)
		res := d.DB.WithContext(ctx).Unscoped().Where("received_at < ?", cutoff).Delete(&model.Event{})
		if res.Error != nil {
			return fmt.Errorf("delete old events: %w", res.Error)
		}
		d.Log.Info("event cleanup complete", "deleted", res.RowsAffected)
		return nil
	}
}

// heartbeatCleanup hard-deletes monitor heartbeats older than the retention window.
func heartbeatCleanup(d JobDeps) func(context.Context) error {
	return func(ctx context.Context) error {
		retention := d.HeartbeatRetention
		if retention <= 0 {
			retention = 30 * 24 * time.Hour
		}
		cutoff := time.Now().UTC().Add(-retention)
		res := d.DB.WithContext(ctx).Unscoped().Where("checked_at < ?", cutoff).Delete(&model.MonitorHeartbeat{})
		if res.Error != nil {
			return fmt.Errorf("delete old heartbeats: %w", res.Error)
		}
		d.Log.Info("heartbeat cleanup complete", "deleted", res.RowsAffected)
		return nil
	}
}
