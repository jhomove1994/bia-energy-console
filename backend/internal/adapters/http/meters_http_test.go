package httpapi

import (
	"encoding/json"
	"math"
	"net/http/httptest"
	"testing"
	"time"
)

func (f *fixture) get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	return do(f.Handler, "GET", path, "", f.Token)
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	return v
}

type listResp struct {
	Items []struct {
		MeterID       string   `json:"meter_id"`
		Status        string   `json:"status"`
		TotalKWh      float64  `json:"total_kwh"`
		VariationPct  *float64 `json:"variation_pct"`
		LastReadingAt string   `json:"last_reading_at"`
	} `json:"items"`
}

func TestListMetersReturnsAllSeededMeters(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, "/api/meters")
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	got := decode[listResp](t, rec)
	if len(got.Items) != 12 || got.Items[0].MeterID != "M-101" || got.Items[11].MeterID != "M-112" {
		t.Fatalf("items = %d", len(got.Items))
	}
	for _, it := range got.Items {
		if it.Status != "OK" || it.TotalKWh <= 0 || it.LastReadingAt != "2026-09-14T23:00:00Z" {
			t.Fatalf("row = %+v", it)
		}
	}

	// No name or location fields are ever exposed.
	raw := decode[map[string][]map[string]any](t, rec)
	for _, banned := range []string{"name", "location"} {
		if _, ok := raw["items"][0][banned]; ok {
			t.Fatalf("meter row exposes %q", banned)
		}
	}
}

func TestListMetersFilterSortAndValidation(t *testing.T) {
	f := newFixture(t)
	ids := func(path string) []string {
		t.Helper()
		rec := f.get(t, path)
		if rec.Code != 200 {
			t.Fatalf("%s status = %d body %s", path, rec.Code, rec.Body)
		}
		var out []string
		for _, it := range decode[listResp](t, rec).Items {
			out = append(out, it.MeterID)
		}
		return out
	}
	if got := ids("/api/meters?q=m-10"); len(got) != 9 || got[0] != "M-101" || got[8] != "M-109" {
		t.Fatalf("q=m-10 -> %v", got)
	}
	if got := ids("/api/meters?search=112"); len(got) != 1 || got[0] != "M-112" {
		t.Fatalf("search alias -> %v", got)
	}
	if got := ids("/api/meters?sort=variation&order=desc"); got[0] != "M-109" {
		t.Fatalf("sort by variation desc starts with %s, want M-109", got[0])
	}
	if got := ids("/api/meters?sort=meter_id&order=desc"); got[0] != "M-112" {
		t.Fatalf("meter_id desc starts with %s", got[0])
	}
	if got := ids("/api/meters?status=Critical"); len(got) != 0 {
		t.Fatalf("status=Critical before analysis -> %v", got)
	}

	for _, bad := range []string{"status=BROKEN", "sort=name", "order=up"} {
		rec := f.get(t, "/api/meters?"+bad)
		if rec.Code != 400 || errorCode(t, rec) != "invalid_query" {
			t.Fatalf("%s -> %d %s, want 400 invalid_query", bad, rec.Code, rec.Body)
		}
	}
}

func TestMeterDetail(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, "/api/meters/M-109")
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	d := decode[struct {
		MeterID      string   `json:"meter_id"`
		Status       string   `json:"status"`
		VariationPct *float64 `json:"variation_pct"`
		AnomalyCount *int     `json:"anomaly_count"`
	}](t, rec)
	if d.MeterID != "M-109" || d.Status != "OK" || d.AnomalyCount == nil || *d.AnomalyCount != 0 {
		t.Fatalf("detail = %+v", d)
	}
	if d.VariationPct == nil || math.Abs(*d.VariationPct-109) > 1 {
		t.Fatalf("variation = %v, want about +109", d.VariationPct)
	}

	rec = f.get(t, "/api/meters/M-999")
	if rec.Code != 404 || errorCode(t, rec) != "not_found" {
		t.Fatalf("unknown meter -> %d %s", rec.Code, rec.Body)
	}
}

type readingsResp struct {
	MeterID string `json:"meter_id"`
	Points  []struct {
		Timestamp   time.Time `json:"timestamp"`
		Consumption float64   `json:"consumption_kwh"`
		Status      string    `json:"status"`
		BaselineKWh *float64  `json:"baseline_kwh"`
		BaselineV   float64   `json:"baseline_voltage_v"`
		BaselineA   float64   `json:"baseline_current_a"`
		BaselinePF  float64   `json:"baseline_power_factor"`
	} `json:"points"`
}

func TestMeterReadingsFullSeriesAndRange(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, "/api/meters/M-101/readings")
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	got := decode[readingsResp](t, rec)
	if got.MeterID != "M-101" || len(got.Points) != 336 {
		t.Fatalf("meter=%s points=%d, want 336", got.MeterID, len(got.Points))
	}
	for i, p := range got.Points {
		if p.BaselineKWh == nil {
			t.Fatalf("point %d has no baseline_kwh", i)
		}
		if i > 0 && !p.Timestamp.After(got.Points[i-1].Timestamp) {
			t.Fatal("points not ascending")
		}
	}
	p0 := got.Points[1] // hour 1 -> 10 + 1%4 = 11
	if *p0.BaselineKWh != 11 || p0.BaselineV != 230 || p0.BaselineA != 100 || p0.BaselinePF != 0.9 {
		t.Fatalf("baseline fields = %+v", p0)
	}

	rec = f.get(t, "/api/meters/M-101/readings?from=2026-09-05T00:00:00Z&to=2026-09-05T23:00:00Z")
	day := decode[readingsResp](t, rec)
	if rec.Code != 200 || len(day.Points) != 24 {
		t.Fatalf("range -> %d with %d points, want 24", rec.Code, len(day.Points))
	}
}

func TestMeterReadingsErrors(t *testing.T) {
	f := newFixture(t)
	tests := []struct {
		name, path string
		status     int
		code       string
	}{
		{"unknown meter", "/api/meters/M-999/readings", 404, "not_found"},
		{"inverted range", "/api/meters/M-101/readings?from=2026-09-06T00:00:00Z&to=2026-09-05T00:00:00Z", 400, "invalid_query"},
		{"unparsable from", "/api/meters/M-101/readings?from=yesterday", 400, "invalid_query"},
		{"unparsable to", "/api/meters/M-101/readings?to=2026-09-05", 400, "invalid_query"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := f.get(t, tt.path)
			if rec.Code != tt.status || errorCode(t, rec) != tt.code {
				t.Fatalf("%s -> %d %s", tt.path, rec.Code, rec.Body)
			}
		})
	}
}

func TestMeterEvents(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, "/api/meters/M-109/events")
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	got := decode[struct {
		Items []struct {
			ID          int64     `json:"id"`
			Timestamp   time.Time `json:"timestamp"`
			Type        string    `json:"type"`
			Description string    `json:"description"`
		} `json:"items"`
	}](t, rec)
	if len(got.Items) != 1 || got.Items[0].Type != "UNKNOWN" || got.Items[0].Description != "firmware update" ||
		!got.Items[0].Timestamp.Equal(fixtureStart.AddDate(0, 0, 12).Add(14*time.Hour)) {
		t.Fatalf("events = %+v", got.Items)
	}

	rec = f.get(t, "/api/meters/M-101/events")
	if rec.Code != 200 || rec.Body.String() != "{\"items\":[]}\n" {
		t.Fatalf("meter without events -> %d %q, want an empty items array", rec.Code, rec.Body)
	}
	if rec = f.get(t, "/api/meters/M-999/events"); rec.Code != 404 {
		t.Fatalf("unknown meter events -> %d", rec.Code)
	}
}

func TestMeterRoutesRequireAuth(t *testing.T) {
	f := newFixture(t)
	for _, path := range []string{"/api/meters", "/api/meters/M-101", "/api/meters/M-101/readings", "/api/meters/M-101/events"} {
		if rec := do(f.Handler, "GET", path, "", ""); rec.Code != 401 {
			t.Fatalf("%s without token -> %d, want 401", path, rec.Code)
		}
	}
}

func TestDashboardSummaryBeforeAnalysis(t *testing.T) {
	f := newFixture(t)
	rec := f.get(t, "/api/dashboard/summary")
	if rec.Code != 200 {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
	raw := decode[map[string]any](t, rec)
	for _, nullable := range []string{"aggregate_confidence", "last_analysis", "summary_text"} {
		v, present := raw[nullable]
		if !present || v != nil {
			t.Fatalf("%s = %v (present=%v), want present and null", nullable, v, present)
		}
	}
	got := decode[struct {
		MetersCount       int     `json:"meters_count"`
		TotalKWh          float64 `json:"total_kwh"`
		AnomaliesCount    int     `json:"anomalies_count"`
		HighPriorityCount int     `json:"high_priority_count"`
	}](t, rec)
	// 12 meters x 14 days, plus the extra kWh of M-109's elevated last day.
	perDay := 276.0 // sum over 24 hours of 10 + hour%4
	want := 12*14*perDay + 1.09*perDay
	if got.MetersCount != 12 || math.Abs(got.TotalKWh-want) > 1e-6 || got.AnomaliesCount != 0 || got.HighPriorityCount != 0 {
		t.Fatalf("dashboard = %+v, want total %v", got, want)
	}
}

func TestDashboardRequiresAuth(t *testing.T) {
	f := newFixture(t)
	if rec := do(f.Handler, "GET", "/api/dashboard/summary", "", ""); rec.Code != 401 {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
