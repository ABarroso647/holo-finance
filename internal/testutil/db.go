package testutil

import (
	"testing"

	"holo/internal/db"
	dbgen "holo/internal/db/generated"
)

// NewDB returns an in-memory SQLite DB with all migrations applied and a Queries instance.
func NewDB(t *testing.T) *dbgen.Queries {
	t.Helper()
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("testutil.NewDB: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return dbgen.New(database)
}
