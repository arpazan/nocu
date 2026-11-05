package migration

import (
	"time"

	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// createMonitors adds the uptime monitor tables: monitors, monitor_notifications
// and monitor_heartbeats.
//
// The structs below are local copies of shared/model so that future model
// changes never alter this migration. Type names are chosen so GORM derives
// the same table names as the live models.
func createMonitors() *gormigrate.Migration {
	type monitor struct {
		model.Base
		Type                     string         `gorm:"size:16;not null;index"`
		Name                     string         `gorm:"size:255;not null"`
		URL                      string         `gorm:"size:2048;not null"`
		IntervalSeconds          int            `gorm:"not null;default:60"`
		Retries                  int            `gorm:"not null;default:0"`
		TimeoutSeconds           int            `gorm:"not null;default:30"`
		Active                   bool           `gorm:"not null;default:true;index"`
		OwnerID                  *uuid.UUID     `gorm:"type:uuid;index"`
		CertExpiryNotification   bool           `gorm:"not null;default:false"`
		DomainExpiryNotification bool           `gorm:"not null;default:false"`
		MaxRedirects             int            `gorm:"not null;default:10"`
		SaveErrorResponse        bool           `gorm:"not null;default:false"`
		Method                   string         `gorm:"size:8;not null;default:GET"`
		BodyEncoding             string         `gorm:"size:8;not null;default:json"`
		Body                     string         `gorm:"type:text"`
		Headers                  datatypes.JSON `gorm:"type:jsonb"`
		AuthMethod               string         `gorm:"size:8;not null;default:none"`
		BasicUsername            string         `gorm:"size:255"`
		BasicPassword            string         `gorm:"size:255"`
		BearerToken              string         `gorm:"size:2048"`
		LastStatus               string         `gorm:"size:16;not null;default:pending;index"`
		LastCheckedAt            *time.Time
		NextCheckAt              *time.Time `gorm:"index"`
		FailureCount             int        `gorm:"not null;default:0"`
		CertExpiresAt            *time.Time
		DomainExpiresAt          *time.Time
	}
	type monitorNotification struct {
		model.Base
		MonitorID uuid.UUID      `gorm:"type:uuid;not null;index"`
		Monitor   *monitor       `gorm:"constraint:OnDelete:CASCADE"`
		Type      string         `gorm:"size:16;not null"`
		Enabled   bool           `gorm:"not null;default:true"`
		Config    datatypes.JSON `gorm:"type:jsonb;not null"`
	}
	type monitorHeartbeat struct {
		model.Base
		MonitorID  uuid.UUID `gorm:"type:uuid;not null;index:idx_monitor_heartbeats_monitor_time,priority:1"`
		Monitor    *monitor  `gorm:"constraint:OnDelete:CASCADE"`
		Status     string    `gorm:"size:16;not null"`
		LatencyMS  int64     `gorm:"not null;default:0"`
		HTTPStatus int       `gorm:"not null;default:0"`
		Message    string    `gorm:"type:text"`
		ErrorBody  string    `gorm:"type:text"`
		Important  bool      `gorm:"not null;default:false;index"`
		CheckedAt  time.Time `gorm:"not null;index:idx_monitor_heartbeats_monitor_time,priority:2"`
	}

	return &gormigrate.Migration{
		ID: "202611050003_create_monitors",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(&monitor{}, &monitorNotification{}, &monitorHeartbeat{})
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(&monitorHeartbeat{}, &monitorNotification{}, &monitor{})
		},
	}
}
