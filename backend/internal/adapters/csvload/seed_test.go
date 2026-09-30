package csvload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"energy/internal/adapters/sqlite"
)

// writeDataDir writes synthetic readings (hoursPerMeter hourly rows per meter,
// omitting the hours in skip) and the given events into a temp directory.
func writeDataDir(t *testing.T, meters []string, hoursPerMeter int, skip map[string]bool, events string) string {
	t.Helper()
	dir := t.TempDir()
	var b strings.Builder
	b.WriteString(readingsHeader)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	for _, m := range meters {
		for h := 0; h < hoursPerMeter; h++ {
			ts := start.Add(time.Duration(h) * time.Hour)
			if skip[m+"|"+ts.Format("2006-01-02 15:04:05")] {
				continue
			}
			fmt.Fprintf(&b, "%s,%s,%d.5,230.0,100.0,0.95,OK\n", m, ts.Format("2006-01-02 15:04:05"), h)
		}
	}
	mustWrite(t, filepath.Join(dir, "readings.csv"), b.String())
	mustWrite(t, filepath.Join(dir, "events.csv"), "meter_id,event_timestamp,event_type,description\n"+events)
	return dir
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func openDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func count(t *testing.T, db *sqlite.DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(query, args...).Scan(&n); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	return n
}

const twoEvents = "X-1,2026-09-01 10:00,SCHEDULED_OUTAGE,Planned outage\nX-2,2026-09-02 00:00,FIRMWARE_UPDATE,Firmware rollout\n"

func TestSeedIsIdempotent(t *testing.T) {
	db := openDB(t)
	dir := writeDataDir(t, []string{"X-1", "X-2"}, 48, nil, twoEvents)
	var logs bytes.Buffer
	ctx := context.Background()

	first, err := Seed(ctx, db, dir, testLogger(&logs))
	if err != nil {
		t.Fatalf("first Seed: %v", err)
	}
	if first.Meters != 2 || first.Readings != 96 || first.Events != 2 || first.Inserted != 100 {
		t.Errorf("first result = %+v", first)
	}
	second, err := Seed(ctx, db, dir, testLogger(&logs))
	if err != nil {
		t.Fatalf("second Seed: %v", err)
	}
	if second.Inserted != 0 {
		t.Errorf("second seed inserted %d rows, want 0", second.Inserted)
	}
	if n := count(t, db, "SELECT count(*) FROM meters"); n != 2 {
		t.Errorf("meters = %d", n)
	}
	if n := count(t, db, "SELECT count(*) FROM readings"); n != 96 {
		t.Errorf("readings = %d", n)
	}
	if n := count(t, db, "SELECT count(*) FROM events"); n != 2 {
		t.Errorf("events = %d", n)
	}
	if !strings.Contains(logs.String(), "seed: meters=2 readings=96 events=2 (inserted=100)") {
		t.Errorf("summary log line missing:\n%s", logs.String())
	}
	if n := count(t, db, "SELECT count(*) FROM events WHERE type='UNKNOWN' AND description='Firmware rollout'"); n != 1 {
		t.Errorf("unrecognized event type must be stored as UNKNOWN with description, got %d", n)
	}
}

func TestSeedPartialDatabaseOnlyAddsMissingRows(t *testing.T) {
	db := openDB(t)
	ctx := context.Background()
	if _, err := db.Exec("INSERT INTO meters (meter_id, created_at) VALUES ('X-1', '2020-01-01T00:00:00Z')"); err != nil {
		t.Fatal(err)
	}
	// Pre-existing row with different values must not be overwritten.
	if _, err := db.Exec("INSERT INTO readings (meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor, status) VALUES ('X-1','2026-09-01T00:00:00Z',999,1,1,1,'OK')"); err != nil {
		t.Fatal(err)
	}
	dir := writeDataDir(t, []string{"X-1", "X-2"}, 24, nil, twoEvents)
	if _, err := Seed(ctx, db, dir, testLogger(&bytes.Buffer{})); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if n := count(t, db, "SELECT count(*) FROM readings"); n != 48 {
		t.Errorf("readings = %d, want 48", n)
	}
	var kwh float64
	if err := db.QueryRow("SELECT consumption_kwh FROM readings WHERE meter_id='X-1' AND ts='2026-09-01T00:00:00Z'").Scan(&kwh); err != nil || kwh != 999 {
		t.Errorf("existing row changed: kwh=%v err=%v", kwh, err)
	}
	var created string
	if err := db.QueryRow("SELECT created_at FROM meters WHERE meter_id='X-1'").Scan(&created); err != nil || created != "2020-01-01T00:00:00Z" {
		t.Errorf("existing meter changed: %q %v", created, err)
	}
}

func TestSeedMissingReadingsFileNamesTheFile(t *testing.T) {
	db := openDB(t)
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "events.csv"), "meter_id,event_timestamp,event_type,description\n")
	_, err := Seed(context.Background(), db, dir, testLogger(&bytes.Buffer{}))
	if err == nil || !strings.Contains(err.Error(), "readings.csv") {
		t.Fatalf("err = %v, want mention of readings.csv", err)
	}
	if !errors.Is(err, ErrDataMissing) {
		t.Errorf("err should wrap ErrDataMissing: %v", err)
	}
}

func TestSeedReportsGapsAndKeepsStatusWithoutInterpolation(t *testing.T) {
	db := openDB(t)
	var logs bytes.Buffer
	skip := map[string]bool{"X-1|2026-09-01 05:00:00": true}
	dir := writeDataDir(t, []string{"X-1", "X-2"}, 24, skip, "")
	// Append one SENSOR_FAULT reading for X-2 at the next hour.
	f, err := os.OpenFile(filepath.Join(dir, "readings.csv"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("X-2,2026-09-02 00:00:00,1.0,230.0,100.0,0.95,SENSOR_FAULT\n")
	f.Close()

	res, err := Seed(context.Background(), db, dir, testLogger(&logs))
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if res.Gaps["X-1"] != 1 || res.Gaps["X-2"] != 0 {
		t.Errorf("gaps = %v, want X-1:1 X-2:0", res.Gaps)
	}
	if res.NonOK != 1 {
		t.Errorf("NonOK = %d, want 1", res.NonOK)
	}
	if n := count(t, db, "SELECT count(*) FROM readings WHERE meter_id='X-1' AND ts='2026-09-01T05:00:00Z'"); n != 0 {
		t.Error("missing 05:00 must not be filled")
	}
	if n := count(t, db, "SELECT count(*) FROM readings WHERE status='SENSOR_FAULT'"); n != 1 {
		t.Errorf("SENSOR_FAULT rows = %d, want 1", n)
	}
	if !strings.Contains(logs.String(), "meter=X-1") || !strings.Contains(logs.String(), "gaps=1") {
		t.Errorf("gap log for X-1 missing:\n%s", logs.String())
	}
}

func TestSeedMeterTableHasNoInventedAttributes(t *testing.T) {
	db := openDB(t)
	if _, err := Seed(context.Background(), db, writeDataDir(t, []string{"X-1"}, 3, nil, ""), testLogger(&bytes.Buffer{})); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query("SELECT name FROM pragma_table_info('meters')")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	allowed := map[string]bool{"id": true, "meter_id": true, "created_at": true}
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if !allowed[name] {
			t.Errorf("unexpected meters column %q", name)
		}
	}
}
