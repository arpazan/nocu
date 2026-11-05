package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

const source = "scheduler"

// Runner schedules jobs and guarantees each runs on one replica at a time.
type Runner struct {
	db      *gorm.DB
	pub     messaging.Publisher
	log     *slog.Logger
	cron    *cron.Cron
	timeout time.Duration
	jobs    []Job
}

// NewRunner builds a Runner. timeout bounds each job run.
func NewRunner(db *gorm.DB, pub messaging.Publisher, log *slog.Logger, timeout time.Duration) *Runner {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	return &Runner{
		db:      db,
		pub:     pub,
		log:     log.With("component", "scheduler"),
		cron:    cron.New(cron.WithParser(specParser), cron.WithLocation(time.UTC)),
		timeout: timeout,
	}
}

// Register validates and schedules a job.
func (r *Runner) Register(job Job) error {
	if err := job.Validate(); err != nil {
		return err
	}
	for _, existing := range r.jobs {
		if existing.Name == job.Name {
			return fmt.Errorf("scheduler: duplicate job %q", job.Name)
		}
	}
	if _, err := r.cron.AddFunc(job.Spec, func() { r.execute(context.Background(), job) }); err != nil {
		return fmt.Errorf("scheduler: schedule %q: %w", job.Name, err)
	}
	r.jobs = append(r.jobs, job)
	r.log.Info("job registered", "job", job.Name, "spec", job.Spec)
	return nil
}

// Jobs lists the registered jobs.
func (r *Runner) Jobs() []Job { return append([]Job(nil), r.jobs...) }

// Start runs the scheduler until ctx is cancelled, then waits for in-flight jobs.
func (r *Runner) Start(ctx context.Context) error {
	if err := r.ensureRows(ctx); err != nil {
		return err
	}
	r.cron.Start()
	r.log.Info("scheduler started", "jobs", len(r.jobs))
	<-ctx.Done()

	stopCtx := r.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-time.After(r.timeout):
		r.log.Warn("jobs still running at shutdown deadline")
	}
	r.log.Info("scheduler stopped")
	return nil
}

// ensureRows creates a scheduled_jobs row per job so the table always reflects the registry.
func (r *Runner) ensureRows(ctx context.Context) error {
	if r.db == nil {
		return nil
	}
	for _, job := range r.jobs {
		row := model.ScheduledJob{Name: job.Name, CronExpr: job.Spec}
		err := r.db.WithContext(ctx).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "name"}},
				DoUpdates: clause.AssignmentColumns([]string{"cron_expr", "updated_at"}),
			}).
			Create(&row).Error
		if err != nil {
			return fmt.Errorf("scheduler: ensure row %q: %w", job.Name, err)
		}
	}
	return nil
}

// execute runs one job under a Postgres advisory lock, records the outcome and publishes it.
func (r *Runner) execute(parent context.Context, job Job) {
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	log := r.log.With("job", job.Name)

	if r.db == nil {
		r.record(ctx, log, job, r.runGuarded(ctx, job))
		return
	}

	// Connection pins one pooled connection so the session-level lock and
	// unlock happen on the same connection.
	err := r.db.WithContext(ctx).Connection(func(tx *gorm.DB) error {
		var locked bool
		if err := tx.Raw("SELECT pg_try_advisory_lock(?)", lockKey(job.Name)).Scan(&locked).Error; err != nil {
			return fmt.Errorf("acquire lock: %w", err)
		}
		if !locked {
			log.Info("skipped, another replica holds the lock")
			return r.persist(ctx, tx, job, outcome{status: model.JobSkipped})
		}
		defer func() {
			if err := tx.Exec("SELECT pg_advisory_unlock(?)", lockKey(job.Name)).Error; err != nil {
				log.Warn("release lock failed", "err", err)
			}
		}()

		out := r.runGuarded(ctx, job)
		r.publish(ctx, job, out)
		return r.persist(ctx, tx, job, out)
	})
	if err != nil {
		log.Error("job bookkeeping failed", "err", err)
	}
}

type outcome struct {
	status   model.JobStatus
	err      error
	duration time.Duration
	ranAt    time.Time
}

// runGuarded executes the job, converting panics into failures.
func (r *Runner) runGuarded(ctx context.Context, job Job) (out outcome) {
	out.ranAt = time.Now().UTC()
	defer func() {
		if rec := recover(); rec != nil {
			out.err = fmt.Errorf("panic: %v", rec)
		}
		out.duration = time.Since(out.ranAt)
		if out.err != nil {
			out.status = model.JobFailed
			r.log.Error("job failed", "job", job.Name, "duration", out.duration, "err", out.err)
		} else {
			out.status = model.JobSucceeded
			r.log.Info("job succeeded", "job", job.Name, "duration", out.duration)
		}
	}()
	out.err = job.Run(ctx)
	return out
}

func (r *Runner) persist(ctx context.Context, tx *gorm.DB, job Job, out outcome) error {
	updates := map[string]any{
		"last_status": out.status,
		"updated_at":  time.Now().UTC(),
	}
	if out.status != model.JobSkipped {
		updates["last_run_at"] = out.ranAt
		updates["last_duration_ms"] = out.duration.Milliseconds()
		updates["run_count"] = gorm.Expr("run_count + 1")
		updates["last_error"] = ""
		if out.err != nil {
			updates["last_error"] = out.err.Error()
		}
	}
	err := tx.WithContext(ctx).Model(&model.ScheduledJob{}).Where("name = ?", job.Name).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("record run: %w", err)
	}
	return nil
}

// record is the no-database path used in tests.
func (r *Runner) record(ctx context.Context, _ *slog.Logger, job Job, out outcome) {
	r.publish(ctx, job, out)
}

func (r *Runner) publish(ctx context.Context, job Job, out outcome) {
	if r.pub == nil {
		return
	}
	subject := messaging.SubjectJobCompleted
	body := map[string]any{
		"job":         job.Name,
		"status":      out.status,
		"ran_at":      out.ranAt,
		"duration_ms": out.duration.Milliseconds(),
	}
	if out.err != nil {
		subject = messaging.SubjectJobFailed
		body["error"] = out.err.Error()
	}
	if err := messaging.PublishJSON(ctx, r.pub, subject, source, body); err != nil {
		r.log.Error("publish failed", "job", job.Name, "err", err)
	}
}
