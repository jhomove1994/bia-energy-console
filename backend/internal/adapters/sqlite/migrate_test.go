package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrateCreatesSchema(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	var version int
	if err := db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil || version != 1 {
		t.Fatalf("user_version = %d (err %v), want 1", version, err)
	}
	for _, table := range []string{"meters", "readings", "events", "analyses", "anomalies"} {
		var n int
		err := db.QueryRowContext(ctx, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n)
		if err != nil || n != 1 {
			t.Errorf("table %s missing (n=%d, err=%v)", table, n, err)
		}
	}
	var fk int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil || fk != 1 {
		t.Errorf("foreign_keys = %d (err %v), want 1", fk, err)
	}
	var mode string
	if err := db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil || mode != "wal" {
		t.Errorf("journal_mode = %q (err %v), want wal", mode, err)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO meters (meter_id, created_at) VALUES ('X-1', '2026-01-01T00:00:00Z')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM meters").Scan(&n); err != nil || n != 1 {
		t.Errorf("meters = %d (err %v), want 1 after second Migrate", n, err)
	}
}
