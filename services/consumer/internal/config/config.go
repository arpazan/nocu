// Package config is the consumer service configuration.
package config

import "github.com/Pzquzer/Nocu/shared/config"

// Config composes the shared blocks the consumer service needs.
type Config struct {
	config.Base
	// DurableName identifies the JetStream consumer shared by every replica.
	DurableName string `env:"CONSUMER_DURABLE_NAME" envDefault:"nocu-consumer"`
	// Subjects lists the subjects to persist, comma separated.
	Subjects []string `env:"CONSUMER_SUBJECTS" envDefault:"incident.>,job.>,monitor.>" envSeparator:","`
}
