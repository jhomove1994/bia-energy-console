package csvload

import (
	"bytes"
	"context"
	"testing"
)

// TestDemoDatasetSeed seeds the demo CSVs and asserts the documented counts
// and a clean cadence, using SQL counts only.
func TestDemoDatasetSeed(t *testing.T) {
	db := openDB(t)
	var logs bytes.Buffer
	res, err := Seed(context.Background(), db, "../../../../data", testLogger(&logs))
	if err != nil {
		t.Fatalf("Seed demo dataset: %v", err)
	}
	if n := count(t, db, "SELECT count(*) FROM meters"); n != 12 {
		t.Errorf("meters = %d, want 12", n)
	}
	if n := count(t, db, "SELECT count(*) FROM readings"); n != 4032 {
		t.Errorf("readings = %d, want 4032", n)
	}
	if n := count(t, db, "SELECT count(*) FROM events"); n != 4 {
		t.Errorf("events = %d, want 4", n)
	}
	if n := count(t, db, "SELECT count(*) FROM readings WHERE status <> 'OK'"); n != 0 {
		t.Errorf("non-OK readings = %d, want 0", n)
	}
	for meter, g := range res.Gaps {
		if g != 0 {
			t.Errorf("meter %s has %d gaps, want 0", meter, g)
		}
	}
	if res.Skipped != 0 {
		t.Errorf("skipped rows = %d, want 0", res.Skipped)
	}
}
