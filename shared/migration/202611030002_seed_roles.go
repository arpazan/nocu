package migration

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Pzquzer/Nocu/shared/model"
)

// seedRoles inserts the three built-in roles.
func seedRoles() *gormigrate.Migration {
	names := []string{model.RoleAdmin, model.RoleOperator, model.RoleViewer}

	return &gormigrate.Migration{
		ID: "202611030002_seed_roles",
		Migrate: func(tx *gorm.DB) error {
			// Local copy so a future change to model.Role cannot alter this migration.
			type role struct {
				model.Base
				Name        string `gorm:"size:64;uniqueIndex;not null"`
				Description string `gorm:"size:255"`
			}
			rows := []role{
				{Name: model.RoleAdmin, Description: "Full access to every resource"},
				{Name: model.RoleOperator, Description: "Can manage incidents"},
				{Name: model.RoleViewer, Description: "Read-only access"},
			}
			return tx.Table("roles").
				Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "name"}}, DoNothing: true}).
				Create(&rows).Error
		},
		Rollback: func(tx *gorm.DB) error {
			return tx.Table("roles").Unscoped().Where("name IN ?", names).Delete(nil).Error
		},
	}
}
