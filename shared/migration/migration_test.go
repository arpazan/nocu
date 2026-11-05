package migration

import (
	"sort"
	"testing"
)

func TestMigrationIDsAreUniqueAndOrdered(t *testing.T) {
	ids := IDs()
	if len(ids) == 0 {
		t.Fatal("no migrations registered")
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate migration id %q", id)
		}
		seen[id] = true
	}
	if !sort.StringsAreSorted(ids) {
		t.Fatalf("migration ids must be sorted ascending: %v", ids)
	}
}
