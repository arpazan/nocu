package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"

	"github.com/Pzquzer/Nocu/shared/model"
)

// initSchema creates every table from the live model definitions.
//
// This is the only migration allowed to reference shared/model directly. Later
// migrations must declare local copies of the structs they touch so that
// changing a model never rewrites history.
func initSchema() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "202611030001_init_schema",
		Migrate: func(tx *gorm.DB) error {
			return tx.AutoMigrate(model.All()...)
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Migrator().DropTable(
				"user_roles",
				&model.ScheduledJob{},
				&model.Event{},
				&model.Incident{},
				&model.User{},
				&model.Role{},
			)
		},
	}
}
