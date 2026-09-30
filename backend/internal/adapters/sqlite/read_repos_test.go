package sqlite

import (
	"context"
	"math"
	"testing"
	"time"

	"energy/internal/domain"
)

var day1 = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// seedSynthetic stores two meters: A-1 with 48 hourly readings (kWh = 1 per
// hour) written in reverse order, and B-2 with 3 readings of 10 kWh.
func seedSynthetic(t *testing.T) *DB {
	t.Helper()
	db := openTemp(t)
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	var rs []domain.Reading
	for i := 47; i >= 0; i-- {
		rs = append(rs, domain.Reading{MeterID: "A-1", TS: day1.Add(time.Duration(i) * time.Hour), KWh: 1, VoltageV: 230, CurrentA: 10, PF: 0.9, Status: "OK"})
	}
	for i := 0; i < 3; i++ {
		rs = append(rs, domain.Reading{MeterID: "B-2", TS: day1.Add(time.Duration(i) * time.Hour), KWh: 10, VoltageV: 231, CurrentA: 11, PF: 0.95, Status: "SENSOR_FAULT"})
	}
	events := []domain.Event{
		{MeterID: "A-1", TS: day1.Add(30 * time.Hour), Type: domain.EventUnknown, Description: "late"},
		{MeterID: "A-1", TS: day1.Add(5 * time.Hour), Type: domain.EventOperationalChange, Description: "early"},
		{MeterID: "B-2", TS: day1.Add(1 * time.Hour), Type: domain.EventDataQuality, Description: "other meter"},
	}
	err := db.WriteTx(ctx, func(w *Writer) error {
		if _, err := w.InsertMeters(ctx, []string{"B-2", "A-1"}, day1); err != nil {
			return err
		}
		if _, err := w.InsertReadings(ctx, rs); err != nil {
			return err
		}
		_, err := w.InsertEvents(ctx, events)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMeterStoreListAndExists(t *testing.T) {
	db := seedSynthetic(t)
	ctx := context.Background()
	s := NewMeterStore(db)

	ms, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 || ms[0].MeterID != "A-1" || ms[1].MeterID != "B-2" {
		t.Fatalf("List = %+v, want A-1 then B-2", ms)
	}
	if !ms[0].CreatedAt.Equal(day1) {
		t.Fatalf("CreatedAt = %v, want %v", ms[0].CreatedAt, day1)
	}
	for id, want := range map[string]bool{"A-1": true, "M-999": false} {
		got, err := s.Exists(ctx, id)
		if err != nil || got != want {
			t.Fatalf("Exists(%s) = %v, %v; want %v", id, got, err, want)
		}
	}
}

func TestReadingStoreByMeter(t *testing.T) {
	db := seedSynthetic(t)
	ctx := context.Background()
	s := NewReadingStore(db)

	all, err := s.ByMeter(ctx, "A-1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 48 {
		t.Fatalf("len = %d, want 48", len(all))
	}
	for i := 1; i < len(all); i++ {
		if !all[i].TS.After(all[i-1].TS) {
			t.Fatalf("not ascending at %d", i)
		}
	}
	if all[0].VoltageV != 230 || all[0].PF != 0.9 || all[0].Status != "OK" {
		t.Fatalf("fields not mapped: %+v", all[0])
	}

	from, to := day1.Add(24*time.Hour), day1.Add(29*time.Hour)
	ranged, err := s.ByMeter(ctx, "A-1", &from, &to)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranged) != 6 || !ranged[0].TS.Equal(from) || !ranged[5].TS.Equal(to) {
		t.Fatalf("inclusive range returned %d rows", len(ranged))
	}

	fromOnly, err := s.ByMeter(ctx, "A-1", &to, nil)
	if err != nil || len(fromOnly) != 19 {
		t.Fatalf("from-only = %d rows, err %v; want 19", len(fromOnly), err)
	}
	none, err := s.ByMeter(ctx, "M-999", nil, nil)
	if err != nil || len(none) != 0 {
		t.Fatalf("unknown meter = %d rows, err %v", len(none), err)
	}
}

func TestReadingStoreAllByMeterAndTotal(t *testing.T) {
	db := seedSynthetic(t)
	ctx := context.Background()
	s := NewReadingStore(db)

	m, err := s.AllByMeter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 || len(m["A-1"]) != 48 || len(m["B-2"]) != 3 {
		t.Fatalf("AllByMeter sizes: A-1=%d B-2=%d meters=%d", len(m["A-1"]), len(m["B-2"]), len(m))
	}
	if !m["A-1"][0].TS.Equal(day1) || !m["A-1"][47].TS.Equal(day1.Add(47*time.Hour)) {
		t.Fatal("A-1 series not sorted by timestamp")
	}
	total, err := s.TotalKWh(ctx)
	if err != nil || math.Abs(total-78) > 1e-9 {
		t.Fatalf("TotalKWh = %v, %v; want 78", total, err)
	}
}

func TestEventStore(t *testing.T) {
	db := seedSynthetic(t)
	ctx := context.Background()
	s := NewEventStore(db)

	all, err := s.All(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("All = %d, %v; want 3", len(all), err)
	}
	a, err := s.ByMeter(ctx, "A-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(a) != 2 || a[0].Description != "early" || a[1].Description != "late" {
		t.Fatalf("ByMeter A-1 = %+v, want early then late", a)
	}
	if a[0].Type != domain.EventOperationalChange || a[0].ID == 0 || !a[0].TS.Equal(day1.Add(5*time.Hour)) {
		t.Fatalf("event fields not mapped: %+v", a[0])
	}
	none, err := s.ByMeter(ctx, "M-999")
	if err != nil || len(none) != 0 {
		t.Fatalf("unknown meter events = %d, %v", len(none), err)
	}
}
