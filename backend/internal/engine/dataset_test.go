package engine_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"energy/internal/adapters/csvload"
	"energy/internal/domain"
	"energy/internal/engine"
)

// The dataset tests run the whole engine on the demo CSV files. This file is
// the only place where meter ids and dates of the dataset appear.

const dataDir = "../../../data"

func loadDataset(t *testing.T) engine.Input {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	rf, err := os.Open(filepath.Join(dataDir, "readings.csv"))
	if err != nil {
		t.Fatalf("readings: %v", err)
	}
	defer rf.Close()
	readings, _, err := csvload.ParseReadings(rf, "readings.csv", log)
	if err != nil {
		t.Fatalf("parse readings: %v", err)
	}

	ef, err := os.Open(filepath.Join(dataDir, "events.csv"))
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	defer ef.Close()
	events, _, err := csvload.ParseEvents(ef, "events.csv", log)
	if err != nil {
		t.Fatalf("parse events: %v", err)
	}

	in := engine.Input{Series: map[string][]domain.Reading{}, Events: events}
	for _, r := range readings {
		in.Series[r.MeterID] = append(in.Series[r.MeterID], r)
	}
	return in
}

func mapMeters(in engine.Input, rename func(string) string, shift time.Duration, keep func(domain.Event) bool) engine.Input {
	out := engine.Input{Series: map[string][]domain.Reading{}}
	for id, series := range in.Series {
		cp := make([]domain.Reading, len(series))
		for i, r := range series {
			r.MeterID = rename(r.MeterID)
			r.TS = r.TS.Add(shift)
			cp[i] = r
		}
		out.Series[rename(id)] = cp
	}
	for _, e := range in.Events {
		if !keep(e) {
			continue
		}
		e.MeterID = rename(e.MeterID)
		e.TS = e.TS.Add(shift)
		out.Events = append(out.Events, e)
	}
	return out
}

func identity(s string) string  { return s }
func keepAll(domain.Event) bool { return true }
func find(res engine.Result, meter string) *domain.Anomaly {
	for i := range res.Anomalies {
		if res.Anomalies[i].MeterID == meter {
			return &res.Anomalies[i]
		}
	}
	return nil
}

func meterIDs(res engine.Result) []string {
	var ids []string
	for _, a := range res.Anomalies {
		ids = append(ids, a.MeterID)
	}
	return ids
}

func TestDatasetYieldsTheFourExpectedAnomaliesInRankOrder(t *testing.T) {
	res := engine.Analyze(loadDataset(t), engine.DefaultConfig(), nil)

	if got, want := meterIDs(res), []string{"M-109", "M-112", "M-104", "M-106"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("anomalies ranked %v, want %v", got, want)
	}
	if res.MetersAnalyzed != 12 {
		t.Fatalf("MetersAnalyzed = %d, want 12", res.MetersAnalyzed)
	}
	for i, a := range res.Anomalies {
		if a.Rank != i+1 {
			t.Fatalf("%s rank = %d, want %d", a.MeterID, a.Rank, i+1)
		}
	}

	cases := []struct {
		meter string
		typ   domain.AnomalyType
		sev   domain.Severity
		kind  domain.AnomalyKind
		act   domain.ActionCode
	}{
		{"M-109", domain.TypeReal, domain.SeverityHigh, domain.KindLevelShiftUp, domain.ActionInspectLoadIncrease},
		{"M-112", domain.TypeDataQuality, domain.SeverityHigh, domain.KindDataInconsistency, domain.ActionOpenDQTicket},
		{"M-104", domain.TypeExplainable, domain.SeverityMedium, domain.KindLevelShiftUp, domain.ActionUpdateBaseline},
		{"M-106", domain.TypeFalsePos, domain.SeverityLow, domain.KindSustainedDecrease, domain.ActionAcknowledgeExpected},
	}
	for _, tt := range cases {
		a := find(res, tt.meter)
		if a.Type != tt.typ || a.Severity != tt.sev || a.Kind != tt.kind || a.Action != tt.act {
			t.Errorf("%s = %s/%s/%s/%s, want %s/%s/%s/%s", tt.meter, a.Type, a.Severity, a.Kind, a.Action, tt.typ, tt.sev, tt.kind, tt.act)
		}
		if len(a.Evidence) == 0 {
			t.Errorf("%s has no evidence", tt.meter)
		}
	}
}

func TestDatasetEventsAreRelatedCorrectly(t *testing.T) {
	res := engine.Analyze(loadDataset(t), engine.DefaultConfig(), nil)

	// The UNKNOWN event is context only: it never turns M-109 into an explained anomaly.
	if e := find(res, "M-109").RelatedEvent; e == nil || e.Type != domain.EventUnknown {
		t.Errorf("M-109 related event = %+v, want the UNKNOWN entry as context", e)
	}
	if e := find(res, "M-106").RelatedEvent; e == nil || e.Type != domain.EventScheduledOutage {
		t.Errorf("M-106 related event = %+v, want the scheduled outage", e)
	}
	if e := find(res, "M-104").RelatedEvent; e == nil || e.Type != domain.EventOperationalChange {
		t.Errorf("M-104 related event = %+v, want the operational change", e)
	}
	if e := find(res, "M-112").RelatedEvent; e == nil || e.Type != domain.EventDataQuality {
		t.Errorf("M-112 related event = %+v, want the data-quality entry", e)
	}
}

func TestDatasetScoresMatchTheDesignWalkThrough(t *testing.T) {
	res := engine.Analyze(loadDataset(t), engine.DefaultConfig(), nil)
	cases := []struct {
		meter              string
		sevLo, sevHi       float64
		confLo, confHi     float64
		priorityLo, prioHi float64
	}{
		{"M-109", 100, 100, 0.97, 0.98, 97, 98},
		{"M-112", 100, 100, 0.98, 0.98, 88, 88.5},
		{"M-104", 45, 52, 0.83, 0.86, 32, 35},
		{"M-106", 0, 39, 0.85, 0.87, 5, 6},
	}
	for _, tt := range cases {
		a := find(res, tt.meter)
		if a.SeverityScore < tt.sevLo || a.SeverityScore > tt.sevHi {
			t.Errorf("%s severity score %v outside [%v, %v]", tt.meter, a.SeverityScore, tt.sevLo, tt.sevHi)
		}
		if a.Confidence < tt.confLo || a.Confidence > tt.confHi {
			t.Errorf("%s confidence %v outside [%v, %v]", tt.meter, a.Confidence, tt.confLo, tt.confHi)
		}
		if a.PriorityScore < tt.priorityLo || a.PriorityScore > tt.prioHi {
			t.Errorf("%s priority %v outside [%v, %v]", tt.meter, a.PriorityScore, tt.priorityLo, tt.prioHi)
		}
	}
}

func TestDatasetHealthyMetersProduceNoAnomalies(t *testing.T) {
	res := engine.Analyze(loadDataset(t), engine.DefaultConfig(), nil)
	for _, m := range []string{"M-101", "M-102", "M-103", "M-105", "M-107", "M-108", "M-110", "M-111"} {
		if a := find(res, m); a != nil {
			t.Errorf("healthy meter %s produced %s %s", m, a.Type, a.Kind)
		}
	}
	if len(res.Anomalies) != 4 {
		t.Fatalf("anomalies = %d, want exactly 4", len(res.Anomalies))
	}
}

func TestDatasetMeterWithoutItsDataQualityEventIsStillDataQuality(t *testing.T) {
	in := mapMeters(loadDataset(t), identity, 0, func(e domain.Event) bool {
		return domain.NormalizeEventType(string(e.Type)) != domain.EventDataQuality
	})
	res := engine.Analyze(in, engine.DefaultConfig(), nil)

	a := find(res, "M-112")
	if a == nil || a.Type != domain.TypeDataQuality || a.Severity != domain.SeverityHigh {
		t.Fatalf("M-112 without its event = %+v", a)
	}
	if a.RelatedEvent != nil {
		t.Fatalf("no event should be related, got %+v", a.RelatedEvent)
	}
	if len(res.Anomalies) != 4 {
		t.Fatalf("anomalies = %d, want 4", len(res.Anomalies))
	}
}

// finding is what must stay identical when dates or meter ids change.
type finding struct {
	Type     domain.AnomalyType
	Kind     domain.AnomalyKind
	Severity domain.Severity
	Score    float64
	Conf     float64
	Priority float64
	Rank     int
	Action   domain.ActionCode
	Hours    int
}

func findingOf(a domain.Anomaly) finding {
	return finding{a.Type, a.Kind, a.Severity, a.SeverityScore, a.Confidence, a.PriorityScore, a.Rank, a.Action,
		int(a.WindowEnd.Sub(a.WindowStart) / time.Hour)}
}

func TestDatasetShiftedByThirtyDaysGivesTheSameFindings(t *testing.T) {
	in := loadDataset(t)
	orig := engine.Analyze(in, engine.DefaultConfig(), nil)

	shift := 30 * 24 * time.Hour
	moved := engine.Analyze(mapMeters(in, identity, shift, keepAll), engine.DefaultConfig(), nil)

	if len(moved.Anomalies) != len(orig.Anomalies) {
		t.Fatalf("anomalies = %d, want %d", len(moved.Anomalies), len(orig.Anomalies))
	}
	for i, a := range orig.Anomalies {
		b := moved.Anomalies[i]
		if a.MeterID != b.MeterID || findingOf(a) != findingOf(b) {
			t.Errorf("rank %d: %s %+v became %s %+v", i+1, a.MeterID, findingOf(a), b.MeterID, findingOf(b))
		}
		if !b.WindowStart.Equal(a.WindowStart.Add(shift)) || !b.WindowEnd.Equal(a.WindowEnd.Add(shift)) {
			t.Errorf("%s window did not move with the data: %v..%v", a.MeterID, b.WindowStart, b.WindowEnd)
		}
	}
}

func TestDatasetWithEveryMeterRenamedGivesTheSameFindings(t *testing.T) {
	in := loadDataset(t)
	orig := engine.Analyze(in, engine.DefaultConfig(), nil)

	// Rename in the opposite lexical order so a hidden dependency on the ids
	// (an id literal or an id ordering) would change the ranks.
	var ids []string
	for id := range in.Series {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	names := map[string]string{}
	for i, id := range ids {
		names[id] = fmt.Sprintf("unit-%02d", len(ids)-i)
	}
	renamed := engine.Analyze(mapMeters(in, func(s string) string { return names[s] }, 0, keepAll), engine.DefaultConfig(), nil)

	if len(renamed.Anomalies) != len(orig.Anomalies) {
		t.Fatalf("anomalies = %d, want %d", len(renamed.Anomalies), len(orig.Anomalies))
	}
	for i, a := range orig.Anomalies {
		b := renamed.Anomalies[i]
		if names[a.MeterID] != b.MeterID || findingOf(a) != findingOf(b) {
			t.Errorf("rank %d: %s %+v became %s %+v", i+1, a.MeterID, findingOf(a), b.MeterID, findingOf(b))
		}
	}
}

func TestDatasetSummaryCounts(t *testing.T) {
	res := engine.Analyze(loadDataset(t), engine.DefaultConfig(), nil)

	total, attention := len(res.Anomalies), 0
	for _, a := range res.Anomalies {
		if a.Severity == domain.SeverityHigh && (a.Type == domain.TypeReal || a.Type == domain.TypeDataQuality) {
			attention++
		}
	}
	if total != 4 || attention != 2 {
		t.Fatalf("summary = %d total, %d priority attention; want 4 and 2", total, attention)
	}
}

func TestDatasetRunsAreIdentical(t *testing.T) {
	in := loadDataset(t)
	a := engine.Analyze(in, engine.DefaultConfig(), nil)
	b := engine.Analyze(in, engine.DefaultConfig(), nil)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("two runs on the same input differ")
	}
}
