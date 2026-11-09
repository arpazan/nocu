package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// monitorAutoIncident adds the per-monitor auto_create_incident toggle and the
// incidents.monitor_id link the scheduler writes when it opens an incident.
//
// A partial unique index guarantees at most one active (open or acknowledged)
// incident per monitor, so concurrent scheduler replicas cannot double-create.
func monitorAutoIncident() *gormigrate.Migration {
	type monitor struct {
		AutoCreateIncident bool `gorm:"not null;default:false"`
	}
	type incident struct {
		MonitorID *uuid.UUID `gorm:"type:uuid;index"`
	}
	const activeIndex = "idx_incidents_monitor_active"

	return &gormigrate.Migration{
		ID: "202611090004_monitor_auto_incident",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Migrator().AddColumn(&monitor{}, "AutoCreateIncident"); err != nil {
				return err
			}
			if err := tx.Migrator().AddColumn(&incident{}, "MonitorID"); err != nil {
				return err
			}
			if err := tx.Migrator().CreateIndex(&incident{}, "MonitorID"); err != nil {
				return err
			}
			return tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS ` + activeIndex + `
				ON incidents (monitor_id)
				WHERE monitor_id IS NOT NULL AND status IN ('open', 'acknowledged') AND deleted_at IS NULL`).Error
		},
		Rollback: func(tx *gorm.DB) error {
			if err := tx.Exec(`DROP INDEX IF EXISTS ` + activeIndex).Error; err != nil {
				return err
			}
			if err := tx.Migrator().DropIndex(&incident{}, "MonitorID"); err != nil {
				return err
			}
			if err := tx.Migrator().DropColumn(&incident{}, "MonitorID"); err != nil {
				return err
			}
			return tx.Migrator().DropColumn(&monitor{}, "AutoCreateIncident")
		},
	}
}
