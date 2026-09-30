package app

import (
	"context"
	"errors"
	"math"
	"sort"
	"testing"
	"time"

	"energy/internal/domain"
	"energy/internal/engine"
)

var origin = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// fakeStore implements the three read repositories in memory.
type fakeStore struct {
	series map[string][]domain.Reading
	events []domain.Event
}

func (f *fakeStore) List(context.Context) ([]domain.Meter, error) {
	var out []domain.Meter
	for id := range f.series {
		out = append(out, domain.Meter{MeterID: id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MeterID < out[j].MeterID })
	return out, nil
}

func (f *fakeStore) Exists(_ context.Context, id string) (bool, error) {
	_, ok := f.series[id]
	return ok, nil
}

func (f *fakeStore) ByMeter(_ context.Context, id string, from, to *time.Time) ([]domain.Reading, error) {
	var out []domain.Reading
	for _, r := range f.series[id] {
		if (from != nil && r.TS.Before(*from)) || (to != nil && r.TS.After(*to)) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func (f *fakeStore) AllByMeter(context.Context) (map[string][]domain.Reading, error) {
	return f.series, nil
}

func (f *fakeStore) TotalKWh(context.Context) (float64, error) {
	var total float64
	for _, rs := range f.series {
		for _, r := range rs {
			total += r.KWh
		}
	}
	return total, nil
}

func (f *fakeStore) All(context.Context) ([]domain.Event, error) { return f.events, nil }

func (f *fakeStore) ByMeterEvents(id string) []domain.Event {
	var out []domain.Event
	for _, e := range f.events {
		if e.MeterID == id {
			out = append(out, e)
		}
	}
	return out
}

// fakeEvents adapts fakeStore to EventRepository (ByMeter name clash).
type fakeEvents struct{ *fakeStore }

func (f fakeEvents) ByMeter(_ context.Context, id string) ([]domain.Event, error) {
	return f.fakeStore.ByMeterEvents(id), nil
}

// mkSeries builds `days` days of hourly readings for id; kwh(day, hour) gives
// the consumption.
func mkSeries(id string, days int, kwh func(day, hour int) float64) []domain.Reading {
	var out []domain.Reading
	for d := 0; d < days; d++ {
		for h := 0; h < 24; h++ {
			out = append(out, domain.Reading{
				MeterID: id, TS: origin.AddDate(0, 0, d).Add(time.Duration(h) * time.Hour),
				KWh: kwh(d, h), VoltageV: 230, CurrentA: 100, PF: 0.9, Status: domain.StatusOK,
			})
		}
	}
	return out
}

func flat(v float64) func(int, int) float64 { return func(int, int) float64 { return v } }

type staticStatuses map[string]MeterFinding

func (s staticStatuses) Statuses(context.Context) (map[string]MeterFinding, error) { return s, nil }

func newService(store *fakeStore, st MeterStatusSource) *MeterService {
	return NewMeterService(store, store, fakeEvents{store}, st, engine.DefaultConfig())
}

func demoStore() *fakeStore {
	return &fakeStore{series: map[string][]domain.Reading{
		"M-101": mkSeries("M-101", 14, flat(10)), // total 3360, variation 0
		"M-109": mkSeries("M-109", 14, func(d, h int) float64 { // last day x2.09
			if d == 13 {
				return 20.9
			}
			return 10
		}),
		"M-110": mkSeries("M-110", 14, func(d, h int) float64 { // last day x1.02
			if d == 13 {
				return 10.2
			}
			return 10
		}),
	}}
}

func TestListMetersRowFields(t *testing.T) {
	svc := newService(demoStore(), NoAnalysis{})
	rows, err := svc.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].MeterID != "M-101" || rows[1].MeterID != "M-109" {
		t.Fatalf("rows = %+v", rows)
	}
	byID := map[string]MeterRow{}
	for _, r := range rows {
		byID[r.MeterID] = r
		if r.Status != StatusOK {
			t.Fatalf("%s status = %s, want OK with NoAnalysis", r.MeterID, r.Status)
		}
		if r.LastReadingAt == nil || !r.LastReadingAt.Equal(origin.AddDate(0, 0, 13).Add(23*time.Hour)) {
			t.Fatalf("%s last_reading_at = %v", r.MeterID, r.LastReadingAt)
		}
	}
	if math.Abs(byID["M-101"].TotalKWh-3360) > 1e-9 {
		t.Fatalf("total = %v, want 3360", byID["M-101"].TotalKWh)
	}
	if v := byID["M-109"].VariationPct; v == nil || math.Abs(*v-109) > 1e-6 {
		t.Fatalf("tripled-ish day variation = %v, want +109", v)
	}
	if v := byID["M-110"].VariationPct; v == nil || math.Abs(*v-2) > 1e-6 {
		t.Fatalf("healthy variation = %v, want +2", v)
	}
	if v := byID["M-101"].VariationPct; v == nil || math.Abs(*v) > 1e-9 {
		t.Fatalf("flat variation = %v, want 0", v)
	}
}

func TestVariationUsesLatestFullDay(t *testing.T) {
	store := &fakeStore{series: map[string][]domain.Reading{
		"X-1": mkSeries("X-1", 14, func(d, h int) float64 {
			if d == 13 { // partial day (see truncation below) must be ignored
				return 100
			}
			if d == 12 {
				return 11
			}
			return 10
		}),
	}}
	rs := store.series["X-1"]
	store.series["X-1"] = rs[:len(rs)-5] // day 13 now has 19 readings
	rows, err := newService(store, NoAnalysis{}).List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if v := rows[0].VariationPct; v == nil || math.Abs(*v-10) > 1e-6 {
		t.Fatalf("variation = %v, want +10 from the last full day", v)
	}
}

func TestVariationNilWithInsufficientHistory(t *testing.T) {
	store := &fakeStore{series: map[string][]domain.Reading{"X-1": mkSeries("X-1", 3, flat(10))}}
	rows, err := newService(store, NoAnalysis{}).List(context.Background(), ListQuery{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("rows=%d err=%v", len(rows), err)
	}
	if rows[0].VariationPct != nil {
		t.Fatalf("variation = %v, want nil", *rows[0].VariationPct)
	}
}

func TestListFilterSearchStatusSort(t *testing.T) {
	statuses := staticStatuses{
		"M-109": {Status: StatusCritical, AnomalyCount: 1},
		"M-110": {Status: StatusAlert, AnomalyCount: 2},
	}
	svc := newService(demoStore(), statuses)
	ctx := context.Background()
	ids := func(q ListQuery) []string {
		t.Helper()
		rows, err := svc.List(ctx, q)
		if err != nil {
			t.Fatalf("List(%+v): %v", q, err)
		}
		var out []string
		for _, r := range rows {
			out = append(out, r.MeterID)
		}
		return out
	}
	eq := func(got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range got {
			if got[i] != want[i] {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	}

	eq(ids(ListQuery{Q: "m-109"}), "M-109")
	eq(ids(ListQuery{Q: "M-1"}), "M-101", "M-109", "M-110")
	eq(ids(ListQuery{Q: "zzz"}))
	eq(ids(ListQuery{Status: "Critical"}), "M-109")
	eq(ids(ListQuery{Status: "alert"}), "M-110")
	eq(ids(ListQuery{Sort: "variation", Order: "desc"}), "M-109", "M-110", "M-101")
	eq(ids(ListQuery{Sort: "variation"}), "M-101", "M-110", "M-109")
	eq(ids(ListQuery{Sort: "status", Order: "desc"}), "M-109", "M-110", "M-101")
	eq(ids(ListQuery{Sort: "kwh", Order: "desc"}), "M-109", "M-110", "M-101")
	eq(ids(ListQuery{Sort: "meter_id", Order: "desc"}), "M-110", "M-109", "M-101")
}

func TestListRejectsInvalidQueryValues(t *testing.T) {
	svc := newService(demoStore(), NoAnalysis{})
	for _, q := range []ListQuery{{Status: "BROKEN"}, {Sort: "name"}, {Order: "sideways"}} {
		_, err := svc.List(context.Background(), q)
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("List(%+v) err = %v, want *ValidationError", q, err)
		}
	}
}

func TestDetailIncludesFindingsCount(t *testing.T) {
	svc := newService(demoStore(), staticStatuses{"M-109": {Status: StatusCritical, AnomalyCount: 3}})
	d, err := svc.Detail(context.Background(), "M-109")
	if err != nil {
		t.Fatal(err)
	}
	if d.MeterID != "M-109" || d.Status != StatusCritical || d.FindingsCount != 3 || d.VariationPct == nil {
		t.Fatalf("detail = %+v", d)
	}
	other, err := svc.Detail(context.Background(), "M-101")
	if err != nil || other.FindingsCount != 0 || other.Status != StatusOK {
		t.Fatalf("healthy detail = %+v, %v", other, err)
	}
	if _, err := svc.Detail(context.Background(), "M-999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown meter err = %v, want ErrNotFound", err)
	}
}

func TestReadingsCarryBaselineAndRangeFilter(t *testing.T) {
	svc := newService(demoStore(), NoAnalysis{})
	ctx := context.Background()

	pts, err := svc.Readings(ctx, "M-109", nil, nil)
	if err != nil || len(pts) != 336 {
		t.Fatalf("points = %d, err %v; want 336", len(pts), err)
	}
	// The baseline comes from the first week only, so it stays 10 even on the
	// elevated last day.
	last := pts[len(pts)-1]
	if last.KWh != 20.9 || math.Abs(last.Baseline.KWh-10) > 1e-9 {
		t.Fatalf("last point kwh=%v baseline=%v", last.KWh, last.Baseline.KWh)
	}
	if pts[0].Baseline.VoltageV != 230 || pts[0].Baseline.CurrentA != 100 || pts[0].Baseline.PF != 0.9 {
		t.Fatalf("additive baselines missing: %+v", pts[0].Baseline)
	}
	for i := 1; i < len(pts); i++ {
		if !pts[i].TS.After(pts[i-1].TS) {
			t.Fatal("points not ascending")
		}
	}

	from := origin.AddDate(0, 0, 4)
	to := from.Add(23 * time.Hour)
	day, err := svc.Readings(ctx, "M-109", &from, &to)
	if err != nil || len(day) != 24 {
		t.Fatalf("day range = %d, err %v; want 24", len(day), err)
	}
	// A range after the baseline week still uses the baseline of the full series.
	late := origin.AddDate(0, 0, 13)
	lateEnd := late.Add(23 * time.Hour)
	pts, err = svc.Readings(ctx, "M-109", &late, &lateEnd)
	if err != nil || len(pts) != 24 || math.Abs(pts[0].Baseline.KWh-10) > 1e-9 {
		t.Fatalf("late range baseline wrong: n=%d err=%v", len(pts), err)
	}
}

func TestReadingsErrors(t *testing.T) {
	svc := newService(demoStore(), NoAnalysis{})
	ctx := context.Background()
	if _, err := svc.Readings(ctx, "M-999", nil, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown meter err = %v", err)
	}
	from, to := origin.AddDate(0, 0, 5), origin.AddDate(0, 0, 2)
	var ve *ValidationError
	if _, err := svc.Readings(ctx, "M-109", &from, &to); !errors.As(err, &ve) {
		t.Fatalf("inverted range err = %v, want *ValidationError", err)
	}
}

func TestEventsByMeter(t *testing.T) {
	store := demoStore()
	store.events = []domain.Event{
		{ID: 1, MeterID: "M-109", TS: origin, Type: domain.EventUnknown},
		{ID: 2, MeterID: "M-101", TS: origin, Type: domain.EventDataQuality},
	}
	svc := newService(store, NoAnalysis{})
	evs, err := svc.Events(context.Background(), "M-109")
	if err != nil || len(evs) != 1 || evs[0].ID != 1 {
		t.Fatalf("events = %+v, %v", evs, err)
	}
	if _, err := svc.Events(context.Background(), "M-999"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown meter err = %v", err)
	}
}
