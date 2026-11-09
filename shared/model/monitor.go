package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MonitorType is the kind of probe a monitor runs.
type MonitorType string

// MonitorHTTP probes an http:// or https:// URL.
const MonitorHTTP MonitorType = "http"

// Valid reports whether the type is one of the known values.
func (t MonitorType) Valid() bool { return t == MonitorHTTP }

// HTTPMethod is an HTTP request method accepted for probes and webhooks.
type HTTPMethod string

const (
	MethodGET     HTTPMethod = "GET"
	MethodPOST    HTTPMethod = "POST"
	MethodPUT     HTTPMethod = "PUT"
	MethodPATCH   HTTPMethod = "PATCH"
	MethodDELETE  HTTPMethod = "DELETE"
	MethodHEAD    HTTPMethod = "HEAD"
	MethodOPTIONS HTTPMethod = "OPTIONS"
)

// Valid reports whether the method is one of the known values.
func (m HTTPMethod) Valid() bool {
	switch m {
	case MethodGET, MethodPOST, MethodPUT, MethodPATCH, MethodDELETE, MethodHEAD, MethodOPTIONS:
		return true
	}
	return false
}

// BodyEncoding is how the probe request body is encoded.
type BodyEncoding string

const (
	BodyJSON BodyEncoding = "json"
	BodyForm BodyEncoding = "form"
	BodyXML  BodyEncoding = "xml"
)

// Valid reports whether the encoding is one of the known values.
func (b BodyEncoding) Valid() bool {
	switch b {
	case BodyJSON, BodyForm, BodyXML:
		return true
	}
	return false
}

// ContentType is the Content-Type header the encoding implies.
func (b BodyEncoding) ContentType() string {
	switch b {
	case BodyForm:
		return "application/x-www-form-urlencoded"
	case BodyXML:
		return "application/xml"
	default:
		return "application/json"
	}
}

// AuthMethod is how the probe authenticates against the target.
type AuthMethod string

const (
	AuthNone   AuthMethod = "none"
	AuthBasic  AuthMethod = "basic"
	AuthBearer AuthMethod = "bearer"
)

// Valid reports whether the auth method is one of the known values.
func (a AuthMethod) Valid() bool {
	switch a {
	case AuthNone, AuthBasic, AuthBearer:
		return true
	}
	return false
}

// MonitorStatus is the last known probe outcome of a monitor.
type MonitorStatus string

const (
	MonitorPending MonitorStatus = "pending"
	MonitorUp      MonitorStatus = "up"
	MonitorDown    MonitorStatus = "down"
)

// Valid reports whether the status is one of the known values.
func (s MonitorStatus) Valid() bool {
	switch s {
	case MonitorPending, MonitorUp, MonitorDown:
		return true
	}
	return false
}

// Default values applied when a monitor is created without them.
const (
	MonitorDefaultInterval     = 60
	MonitorDefaultTimeout      = 30
	MonitorDefaultMaxRedirects = 10
	MonitorMinInterval         = 20
)

// Monitor is an uptime check configured by an operator.
//
// Secrets (BasicPassword, BearerToken) are never serialised; the JSON form
// exposes has_basic_password and has_bearer_token instead.
type Monitor struct {
	Base

	// General
	Type            MonitorType `gorm:"size:16;not null;index" json:"type"`
	Name            string      `gorm:"size:255;not null" json:"name"`
	URL             string      `gorm:"size:2048;not null" json:"url"`
	IntervalSeconds int         `gorm:"not null;default:60" json:"interval_seconds"`
	Retries         int         `gorm:"not null;default:0" json:"retries"`
	TimeoutSeconds  int         `gorm:"not null;default:30" json:"timeout_seconds"`
	Active          bool        `gorm:"not null;default:true;index" json:"active"`
	OwnerID         *uuid.UUID  `gorm:"type:uuid;index" json:"owner_id,omitempty"`

	// Advanced
	CertExpiryNotification   bool `gorm:"not null;default:false" json:"cert_expiry_notification"`
	DomainExpiryNotification bool `gorm:"not null;default:false" json:"domain_expiry_notification"`
	MaxRedirects             int  `gorm:"not null;default:10" json:"max_redirects"`
	SaveErrorResponse        bool `gorm:"not null;default:false" json:"save_error_response"`
	// AutoCreateIncident opens a critical incident when the monitor flips to down.
	AutoCreateIncident bool `gorm:"not null;default:false" json:"auto_create_incident"`

	// HTTP options
	Method       HTTPMethod     `gorm:"size:8;not null;default:GET" json:"method"`
	BodyEncoding BodyEncoding   `gorm:"size:8;not null;default:json" json:"body_encoding"`
	Body         string         `gorm:"type:text" json:"body"`
	Headers      datatypes.JSON `gorm:"type:jsonb" json:"headers"`

	// Authentication
	AuthMethod    AuthMethod `gorm:"size:8;not null;default:none" json:"auth_method"`
	BasicUsername string     `gorm:"size:255" json:"basic_username,omitempty"`
	BasicPassword string     `gorm:"size:255" json:"-"`
	BearerToken   string     `gorm:"size:2048" json:"-"`

	// Probe state, maintained by the scheduler.
	LastStatus      MonitorStatus `gorm:"size:16;not null;default:pending;index" json:"last_status"`
	LastCheckedAt   *time.Time    `json:"last_checked_at,omitempty"`
	NextCheckAt     *time.Time    `gorm:"index" json:"next_check_at,omitempty"`
	FailureCount    int           `gorm:"not null;default:0" json:"failure_count"`
	CertExpiresAt   *time.Time    `json:"cert_expires_at,omitempty"`
	DomainExpiresAt *time.Time    `json:"domain_expires_at,omitempty"`

	Notifications []MonitorNotification `gorm:"foreignKey:MonitorID;constraint:OnDelete:CASCADE" json:"notifications"`
}

// MarshalJSON adds has_basic_password and has_bearer_token without exposing the values.
func (m Monitor) MarshalJSON() ([]byte, error) {
	type alias Monitor
	return json.Marshal(struct {
		alias
		HasBasicPassword bool `json:"has_basic_password"`
		HasBearerToken   bool `json:"has_bearer_token"`
	}{
		alias:            alias(m),
		HasBasicPassword: m.BasicPassword != "",
		HasBearerToken:   m.BearerToken != "",
	})
}

// HeaderMap decodes the Headers column. A nil or empty column yields an empty map.
func (m *Monitor) HeaderMap() (map[string]string, error) {
	out := map[string]string{}
	if len(m.Headers) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(m.Headers, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Interval is IntervalSeconds as a Duration.
func (m *Monitor) Interval() time.Duration { return time.Duration(m.IntervalSeconds) * time.Second }

// Timeout is TimeoutSeconds as a Duration.
func (m *Monitor) Timeout() time.Duration { return time.Duration(m.TimeoutSeconds) * time.Second }

// MonitorHeartbeat is one probe result.
type MonitorHeartbeat struct {
	Base
	MonitorID  uuid.UUID     `gorm:"type:uuid;not null;index:idx_monitor_heartbeats_monitor_time,priority:1" json:"monitor_id"`
	Status     MonitorStatus `gorm:"size:16;not null" json:"status"`
	LatencyMS  int64         `gorm:"not null;default:0" json:"latency_ms"`
	HTTPStatus int           `gorm:"not null;default:0" json:"http_status"`
	Message    string        `gorm:"type:text" json:"message"`
	// ErrorBody holds the response body of a failed probe when the monitor has SaveErrorResponse set.
	ErrorBody string `gorm:"type:text" json:"error_body,omitempty"`
	// Important marks a heartbeat that changed the monitor status (up -> down or down -> up).
	Important bool      `gorm:"not null;default:false;index" json:"important"`
	CheckedAt time.Time `gorm:"not null;index:idx_monitor_heartbeats_monitor_time,priority:2" json:"checked_at"`
}
