package model

import (
	"time"

	"gorm.io/datatypes"
)

// Event is a message persisted by the consumer service.
type Event struct {
	Base
	Subject    string         `gorm:"size:255;not null;index" json:"subject"`
	Type       string         `gorm:"size:128;not null;index" json:"type"`
	Payload    datatypes.JSON `gorm:"type:jsonb" json:"payload"`
	ReceivedAt time.Time      `gorm:"not null;index" json:"received_at"`
}
