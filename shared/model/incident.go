package model

import (
	"time"

	"github.com/google/uuid"
)

// Severity ranks how urgent an incident is.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// Valid reports whether the severity is one of the known values.
func (s Severity) Valid() bool {
	switch s {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	}
	return false
}

// IncidentStatus is the lifecycle state of an incident.
type IncidentStatus string

const (
	IncidentOpen         IncidentStatus = "open"
	IncidentAcknowledged IncidentStatus = "acknowledged"
	IncidentResolved     IncidentStatus = "resolved"
	IncidentClosed       IncidentStatus = "closed"
)

// Valid reports whether the status is one of the known values.
func (s IncidentStatus) Valid() bool {
	switch s {
	case IncidentOpen, IncidentAcknowledged, IncidentResolved, IncidentClosed:
		return true
	}
	return false
}

// CanTransitionTo enforces open -> acknowledged -> resolved -> closed, with
// closing allowed from any non-closed state.
func (s IncidentStatus) CanTransitionTo(next IncidentStatus) bool {
	if s == next {
		return false
	}
	switch s {
	case IncidentOpen:
		return next == IncidentAcknowledged || next == IncidentResolved || next == IncidentClosed
	case IncidentAcknowledged:
		return next == IncidentResolved || next == IncidentClosed
	case IncidentResolved:
		return next == IncidentClosed || next == IncidentOpen
	case IncidentClosed:
		return false
	}
	return false
}

// Incident is an operational issue tracked by the NOC.
type Incident struct {
	Base
	Title          string         `gorm:"size:255;not null" json:"title"`
	Description    string         `gorm:"type:text" json:"description"`
	Severity       Severity       `gorm:"size:16;not null;index" json:"severity"`
	Status         IncidentStatus `gorm:"size:16;not null;index;default:open" json:"status"`
	Source         string         `gorm:"size:128;index" json:"source"`
	ReporterID     *uuid.UUID     `gorm:"type:uuid;index" json:"reporter_id,omitempty"`
	AcknowledgedAt *time.Time     `json:"acknowledged_at,omitempty"`
	ResolvedAt     *time.Time     `json:"resolved_at,omitempty"`
	ClosedAt       *time.Time     `json:"closed_at,omitempty"`
}
