package model

import "time"

// JobStatus is the outcome of the last run of a scheduled job.
type JobStatus string

const (
	JobSucceeded JobStatus = "succeeded"
	JobFailed    JobStatus = "failed"
	JobSkipped   JobStatus = "skipped"
)

// ScheduledJob tracks the last run of each cron job in the scheduler service.
type ScheduledJob struct {
	Base
	Name           string     `gorm:"size:128;uniqueIndex;not null" json:"name"`
	CronExpr       string     `gorm:"size:64;not null" json:"cron_expr"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastStatus     JobStatus  `gorm:"size:16" json:"last_status"`
	LastError      string     `gorm:"type:text" json:"last_error,omitempty"`
	LastDurationMS int64      `json:"last_duration_ms"`
	RunCount       int64      `gorm:"not null;default:0" json:"run_count"`
}
