// Package config is the scheduler service configuration.
package config

import (
	"time"

	"github.com/Pzquzer/Nocu/shared/config"
)

// Config composes the shared blocks the scheduler service needs.
type Config struct {
	config.Base
	// JobTimeout bounds a single job run.
	JobTimeout time.Duration `env:"SCHEDULER_JOB_TIMEOUT" envDefault:"5m"`
	// SLAAckDeadline is how long a critical incident may stay unacknowledged.
	SLAAckDeadline time.Duration `env:"SCHEDULER_SLA_ACK_DEADLINE" envDefault:"15m"`
	// EventRetention is how long consumer events are kept before cleanup.
	EventRetention time.Duration `env:"SCHEDULER_EVENT_RETENTION" envDefault:"720h"`

	// MonitorTick is how often the uptime worker claims due monitors.
	MonitorTick time.Duration `env:"SCHEDULER_MONITOR_TICK" envDefault:"10s"`
	// MonitorConcurrency bounds monitors claimed and probed per tick.
	MonitorConcurrency int `env:"SCHEDULER_MONITOR_CONCURRENCY" envDefault:"10"`
	// MonitorAllowPrivate lets probes and webhooks reach loopback, private and
	// link-local addresses. Keep it off outside isolated lab setups (SSRF guard).
	MonitorAllowPrivate bool `env:"SCHEDULER_MONITOR_ALLOW_PRIVATE" envDefault:"false"`
	// HeartbeatRetention is how long monitor heartbeats are kept before cleanup.
	HeartbeatRetention time.Duration `env:"SCHEDULER_HEARTBEAT_RETENTION" envDefault:"720h"`
	// ExpiryWarnDays lists the days-left thresholds for certificate and domain expiry warnings.
	ExpiryWarnDays []int `env:"SCHEDULER_EXPIRY_WARN_DAYS" envDefault:"7,14,30" envSeparator:","`
}
