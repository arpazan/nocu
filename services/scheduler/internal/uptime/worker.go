package uptime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/Pzquzer/Nocu/shared/messaging"
	"github.com/Pzquzer/Nocu/shared/model"
)

const source = "scheduler"

// WorkerOptions tunes the probe loop.
type WorkerOptions struct {
	// Tick is how often due monitors are claimed.
	Tick time.Duration
	// Concurrency bounds monitors claimed and probed per tick.
	Concurrency int
}

// Worker claims due monitors on every tick, probes them and records the outcome.
//
// It deliberately does not use the cron Runner's advisory lock: monitor
// intervals (20s minimum) are finer than cron, and SELECT ... FOR UPDATE SKIP
// LOCKED in Store.Due already keeps replicas from probing the same monitor.
type Worker struct {
	store    Store
	prober   *Prober
	pub      messaging.Publisher
	dispatch *Dispatcher
	log      *slog.Logger
	opts     WorkerOptions
	now      func() time.Time
}

// NewWorker wires a Worker.
func NewWorker(store Store, prober *Prober, pub messaging.Publisher, dispatch *Dispatcher, log *slog.Logger, opts WorkerOptions) *Worker {
	if opts.Tick <= 0 {
		opts.Tick = 10 * time.Second
	}
	if opts.Concurrency <= 0 {
		opts.Concurrency = 10
	}
	return &Worker{
		store:    store,
		prober:   prober,
		pub:      pub,
		dispatch: dispatch,
		log:      log.With("component", "uptime"),
		opts:     opts,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// Run loops until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.opts.Tick)
	defer ticker.Stop()
	w.log.Info("uptime worker started", "tick", w.opts.Tick, "concurrency", w.opts.Concurrency)
	for {
		select {
		case <-ctx.Done():
			w.log.Info("uptime worker stopped")
			return nil
		case <-ticker.C:
			w.RunOnce(ctx)
		}
	}
}

// RunOnce claims and probes one batch of due monitors.
func (w *Worker) RunOnce(ctx context.Context) {
	due, err := w.store.Due(ctx, w.now(), w.opts.Concurrency)
	if err != nil {
		w.log.ErrorContext(ctx, "claim due monitors failed", "err", err)
		return
	}
	if len(due) == 0 {
		return
	}
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(w.opts.Concurrency)
	for i := range due {
		m := &due[i]
		g.Go(func() error {
			w.Check(gctx, m)
			return nil
		})
	}
	_ = g.Wait()
}

// Check probes one monitor, applies the state machine, persists the heartbeat
// and fans out events and notifications when the status changed.
func (w *Worker) Check(ctx context.Context, m *model.Monitor) {
	log := w.log.With("monitor_id", m.ID, "monitor", m.Name)
	res := w.prober.Probe(ctx, m)
	hb, fields := transition(m, res, w.now())

	if err := w.store.SaveHeartbeat(ctx, hb); err != nil {
		log.ErrorContext(ctx, "save heartbeat failed", "err", err)
	}
	if err := w.store.UpdateMonitor(ctx, m.ID, fields); err != nil {
		log.ErrorContext(ctx, "update monitor failed", "err", err)
	}
	log.Debug("probe complete", "status", hb.Status, "http_status", hb.HTTPStatus, "latency_ms", hb.LatencyMS, "message", hb.Message)

	if !hb.Important {
		return
	}
	log.Info("monitor status changed", "status", hb.Status, "message", hb.Message)
	ev := Event{
		MonitorID:  m.ID,
		Name:       m.Name,
		URL:        m.URL,
		Status:     hb.Status,
		Message:    hb.Message,
		HTTPStatus: hb.HTTPStatus,
		LatencyMS:  hb.LatencyMS,
		CheckedAt:  hb.CheckedAt,
		Kind:       KindStatus,
	}
	w.publish(ctx, ev)
	if w.dispatch != nil {
		w.dispatch.Dispatch(ctx, m.Notifications, ev)
	}
	if hb.Status == model.MonitorDown && m.AutoCreateIncident {
		w.openIncident(ctx, m, ev)
	}
}

// incidentSource is the Incident.Source value of auto-created incidents.
const incidentSource = "monitor"

// buildIncident shapes the critical incident opened when m flips to down.
func buildIncident(m *model.Monitor, ev Event) *model.Incident {
	id := m.ID
	desc := fmt.Sprintf("Monitor %s (%s) went down at %s.\n\nProbe result: %s",
		m.Name, m.URL, ev.CheckedAt.UTC().Format(time.RFC3339), ev.Message)
	if ev.HTTPStatus > 0 {
		desc += fmt.Sprintf(" (HTTP %d)", ev.HTTPStatus)
	}
	return &model.Incident{
		Title:       ev.Title(),
		Description: desc,
		Severity:    model.SeverityCritical,
		Status:      model.IncidentOpen,
		Source:      incidentSource,
		MonitorID:   &id,
	}
}

// openIncident creates a critical incident for a monitor that just went down,
// unless one is already active, and publishes incident.created. Like publish
// and Dispatch it is best-effort: a failure is logged, never returned, so a
// database hiccup cannot stall probing.
func (w *Worker) openIncident(ctx context.Context, m *model.Monitor, ev Event) {
	log := w.log.With("monitor_id", m.ID, "monitor", m.Name)
	inc := buildIncident(m, ev)
	created, err := w.store.CreateIncidentIfNone(ctx, inc)
	if err != nil {
		log.ErrorContext(ctx, "auto-create incident failed", "err", err)
		return
	}
	if !created {
		log.Info("auto-create incident skipped: one is already active")
		return
	}
	log.Info("incident auto-created", "incident_id", inc.ID, "severity", inc.Severity)
	if w.pub == nil {
		return
	}
	if err := messaging.PublishJSON(ctx, w.pub, messaging.SubjectIncidentCreated, source, inc); err != nil {
		w.log.ErrorContext(ctx, "publish failed", "subject", messaging.SubjectIncidentCreated, "incident_id", inc.ID, "err", err)
	}
}

// transition applies the retry state machine to a probe result. It mutates m's
// in-memory state and returns the heartbeat to insert plus the columns to update.
func transition(m *model.Monitor, res Result, at time.Time) (*model.MonitorHeartbeat, map[string]any) {
	prev := m.LastStatus
	if prev == "" {
		prev = model.MonitorPending
	}
	hb := &model.MonitorHeartbeat{
		MonitorID:  m.ID,
		HTTPStatus: res.HTTPStatus,
		LatencyMS:  res.Latency.Milliseconds(),
		Message:    res.Message,
		ErrorBody:  res.ErrorBody,
		CheckedAt:  at,
	}

	next := prev
	failures := m.FailureCount
	if res.Status == model.MonitorUp {
		failures = 0
		next = model.MonitorUp
		hb.Status = model.MonitorUp
	} else {
		failures++
		if prev != model.MonitorDown && failures <= m.Retries {
			hb.Status = model.MonitorPending
			hb.Message = fmt.Sprintf("retry %d/%d: %s", failures, m.Retries, res.Message)
		} else {
			next = model.MonitorDown
			hb.Status = model.MonitorDown
		}
	}
	hb.Important = next != prev && hb.Status != model.MonitorPending

	fields := map[string]any{
		"last_status":     next,
		"last_checked_at": at,
		"failure_count":   failures,
	}
	if res.CertExpiresAt != nil {
		fields["cert_expires_at"] = *res.CertExpiresAt
		m.CertExpiresAt = res.CertExpiresAt
	}
	m.LastStatus = next
	m.FailureCount = failures
	m.LastCheckedAt = &at
	return hb, fields
}

// publish is best-effort, like the incident service: a broker failure is logged, not returned.
func (w *Worker) publish(ctx context.Context, ev Event) {
	if w.pub == nil {
		return
	}
	subject := messaging.SubjectMonitorUp
	if ev.Status == model.MonitorDown {
		subject = messaging.SubjectMonitorDown
	}
	body := map[string]any{
		"monitor_id":  ev.MonitorID,
		"name":        ev.Name,
		"url":         ev.URL,
		"status":      ev.Status,
		"message":     ev.Message,
		"http_status": ev.HTTPStatus,
		"latency_ms":  ev.LatencyMS,
		"checked_at":  ev.CheckedAt,
	}
	if err := messaging.PublishJSON(ctx, w.pub, subject, source, body); err != nil {
		w.log.ErrorContext(ctx, "publish failed", "subject", subject, "monitor_id", ev.MonitorID, "err", err)
	}
}
