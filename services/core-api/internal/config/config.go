// Package config is the core-api service configuration.
package config

import (
	"time"

	"github.com/Pzquzer/Nocu/shared/config"
)

// Config composes the shared blocks the core-api service needs.
type Config struct {
	config.Base
	config.HTTP
	config.JWT
	// WSWriteTimeout bounds a single WebSocket write.
	WSWriteTimeout time.Duration `env:"WS_WRITE_TIMEOUT" envDefault:"5s"`
	// WSPingInterval is how often the server pings idle clients.
	WSPingInterval time.Duration `env:"WS_PING_INTERVAL" envDefault:"30s"`
	// WSSendBuffer is the per-client outbound queue; slow clients exceeding it are dropped.
	WSSendBuffer int `env:"WS_SEND_BUFFER" envDefault:"64"`
}
