// Package migration holds the versioned schema migrations run by cmd/migrate.
//
// Migrations are ordered by ID and executed with gormigrate inside a
// transaction. Add new migrations at the end of the list; never edit an
// applied one.
package migration

import (
	"fmt"

	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

const tableName = "schema_migrations"

// list is the ordered set of migrations.
var list = []*gormigrate.Migration{
	initSchema(),
	seedRoles(),
	createMonitors(),
}

func newMigrator(db *gorm.DB) *gormigrate.Gormigrate {
	return gormigrate.New(db, &gormigrate.Options{
		TableName:                 tableName,
		IDColumnName:              "id",
		IDColumnSize:              255,
		UseTransaction:            true,
		ValidateUnknownMigrations: true,
	}, list)
}

// Up applies every pending migration.
func Up(db *gorm.DB) error {
	if err := newMigrator(db).Migrate(); err != nil {
		return fmt.Errorf("migration: up: %w", err)
	}
	return nil
}

// Down rolls back the most recently applied migration.
func Down(db *gorm.DB) error {
	if err := newMigrator(db).RollbackLast(); err != nil {
		return fmt.Errorf("migration: down: %w", err)
	}
	return nil
}

// Entry describes one migration and whether it has been applied.
type Entry struct {
	ID      string
	Applied bool
}

// Status reports every known migration and its applied state.
func Status(db *gorm.DB) ([]Entry, error) {
	applied := map[string]bool{}
	if db.Migrator().HasTable(tableName) {
		var ids []string
		if err := db.Table(tableName).Pluck("id", &ids).Error; err != nil {
			return nil, fmt.Errorf("migration: status: %w", err)
		}
		for _, id := range ids {
			applied[id] = true
		}
	}
	entries := make([]Entry, 0, len(list))
	for _, m := range list {
		entries = append(entries, Entry{ID: m.ID, Applied: applied[m.ID]})
	}
	return entries, nil
}

// IDs returns the migration IDs in order. Exposed for tests.
func IDs() []string {
	ids := make([]string, 0, len(list))
	for _, m := range list {
		ids = append(ids, m.ID)
	}
	return ids
}
