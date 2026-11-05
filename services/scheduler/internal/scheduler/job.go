// Package scheduler runs cron jobs with single-instance locking and run tracking.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"

	"github.com/robfig/cron/v3"
)

// Job is a unit of scheduled work.
type Job struct {
	// Name is unique and used for the advisory lock and the scheduled_jobs row.
	Name string
	// Spec is a standard 5-field cron expression, or a descriptor such as "@every 1m".
	Spec string
	Run  func(ctx context.Context) error
}

var specParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// Validate checks the job is well-formed.
func (j Job) Validate() error {
	if j.Name == "" {
		return errors.New("scheduler: job name is required")
	}
	if j.Run == nil {
		return fmt.Errorf("scheduler: job %q has no Run func", j.Name)
	}
	if _, err := specParser.Parse(j.Spec); err != nil {
		return fmt.Errorf("scheduler: job %q has invalid spec %q: %w", j.Name, j.Spec, err)
	}
	return nil
}

// lockKey maps a job name to a stable int64 for pg_try_advisory_lock.
func lockKey(name string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte("nocu:scheduler:" + name))
	return int64(h.Sum64())
}
