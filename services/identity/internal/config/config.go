// Package config is the identity service configuration.
package config

import "github.com/Pzquzer/Nocu/shared/config"

// Config composes the shared blocks the identity service needs.
type Config struct {
	config.Base
	config.HTTP
	config.JWT
	// DefaultRole is granted to every newly registered user.
	DefaultRole string `env:"IDENTITY_DEFAULT_ROLE" envDefault:"viewer"`
	// MinPasswordLength is enforced at registration.
	MinPasswordLength int `env:"IDENTITY_MIN_PASSWORD_LENGTH" envDefault:"8"`
}
