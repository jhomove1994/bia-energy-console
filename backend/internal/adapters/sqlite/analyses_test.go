package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

var t0 = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

// analysisFixture is a migrated database with meters A-1, B-2 and one event on A-1.
func analysisFixture(t *testing.T) (*DB, *AnalysisStore, *AnomalyStore) {
	t.Helper()
	db := openTemp(t)
	ctx := context.Background()
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	err := db.WriteTx(ctx, func(w *Writer) error {
		if _, err := w.InsertMeters(ctx, []string{"A-1", "B-2", "C-3"}, t0); err != nil {
			return err
		}
		_, err := w.InsertEvents(ctx, []domain.Event{{MeterID: "A-1", TS: t0, Type: domain.EventUnknown, Description: "firmware update"}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return db, NewAnalysisStore(db), NewAnomalyStore(db)
}

func newAnalysis(id string, created time.Time) domain.Analysis {
	return domain.Analysis{ID: id, Status: domain.AnalysisPending, CreatedAt: created}
}

func f64(v float64) *float64 { return &v }

func anomaly(id, analysisID, meter string, rank int, typ domain.AnomalyType, sev domain.Severity) domain.Anomaly {
	return domain.Anomaly{
		ID: id, AnalysisID: analysisID, MeterID: meter, DetectedAt: t0, WindowStart: t0, WindowEnd: t0.Add(5 * time.Hour),
		Kind: domain.KindLevelShiftUp, Type: typ, Severity: sev, SeverityScore: 88.5, Confidence: 0.9, PriorityScore: 70.2,
		Rank: rank, Action: domain.ActionInspectLoadIncrease, Reason: "reason " + id, RecommendedAction: "act " + id,
		Evidence: []domain.Evidence{{Metric: "consumption_kwh", Label: "Consumption", Baseline: f64(20), Observed: 42, DeltaPct: f64(110), Unit: "kWh"}},
		Status:   "open",
	}
}

func TestAnalysisLifecycle(t *testing.T) {
	_, s, _ := analysisFixture(t)
	ctx := context.Background()

	if err := s.Create(ctx, newAnalysis("a1", t0)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(ctx, "a1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.AnalysisPending || got.Progress != 0 || !got.CreatedAt.Equal(t0) || got.StartedAt != nil || got.FinishedAt != nil || got.Summary != nil {
		t.Fatalf("created analysis = %+v", got)
	}

	if err := s.UpdateProgress(ctx, "a1", domain.AnalysisRunning, domain.StageDetection, 3, 28); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(ctx, "a1")
	if got.Status != domain.AnalysisRunning || got.Stage != domain.StageDetection || got.StageIndex != 3 || got.Progress != 28 {
		t.Fatalf("after UpdateProgress = %+v", got)
	}
	if got.StartedAt == nil {
		t.Fatal("first running update must set started_at")
	}
	firstStart := *got.StartedAt
	if err := s.UpdateProgress(ctx, "a1", domain.AnalysisRunning, domain.StageEvents, 4, 42); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(ctx, "a1")
	if !got.StartedAt.Equal(firstStart) || got.Progress != 42 {
		t.Fatalf("started_at must not move: %+v", got)
	}

	done := t0.Add(time.Minute)
	sum := domain.Summary{Total: 2, PriorityAttention: 1, MetersAnalyzed: 3, AvgConfidence: 0.9,
		ByType:     map[domain.AnomalyType]int{domain.TypeReal: 1, domain.TypeDataQuality: 1},
		BySeverity: map[domain.Severity]int{domain.SeverityHigh: 2}, Text: "2 anomalies detected · 1 requires priority attention"}
	as := []domain.Anomaly{anomaly("x1", "a1", "A-1", 1, domain.TypeReal, domain.SeverityHigh), anomaly("x2", "a1", "B-2", 2, domain.TypeDataQuality, domain.SeverityHigh)}
	if err := s.Complete(ctx, "a1", done, sum, as); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(ctx, "a1")
	if got.Status != domain.AnalysisDone || got.Progress != 100 || got.FinishedAt == nil || !got.FinishedAt.Equal(done) || got.Error != "" {
		t.Fatalf("after Complete = %+v", got)
	}
	if got.Summary == nil || got.Summary.Total != 2 || got.Summary.PriorityAttention != 1 || got.Summary.Text != sum.Text ||
		got.Summary.ByType[domain.TypeReal] != 1 || got.Summary.BySeverity[domain.SeverityHigh] != 2 || got.Summary.AvgConfidence != 0.9 {
		t.Fatalf("summary round trip = %+v", got.Summary)
	}
}

func TestCompleteIsAtomic(t *testing.T) {
	_, s, an := analysisFixture(t)
	ctx := context.Background()
	if err := s.Create(ctx, newAnalysis("a1", t0)); err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateProgress(ctx, "a1", domain.AnalysisRunning, domain.StageReadings, 1, 0); err != nil {
		t.Fatal(err)
	}
	// The second anomaly references a meter that does not exist: the insert fails.
	as := []domain.Anomaly{anomaly("x1", "a1", "A-1", 1, domain.TypeReal, domain.SeverityHigh), anomaly("x2", "a1", "NOPE", 2, domain.TypeReal, domain.SeverityLow)}
	if err := s.Complete(ctx, "a1", t0, domain.Summary{Total: 2}, as); err == nil {
		t.Fatal("Complete with an invalid anomaly must fail")
	}
	got, _ := s.Get(ctx, "a1")
	if got.Status != domain.AnalysisRunning || got.Progress != 0 || got.Summary != nil {
		t.Fatalf("failed Complete must leave the analysis untouched: %+v", got)
	}
	list, err := an.List(ctx, "a1", app.AnomalyFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("no anomaly may be visible after a failed Complete, got %d", len(list))
	}
	if _, err := an.Get(ctx, "x1"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("Get(x1) = %v, want ErrNotFound", err)
	}
}

func TestFailStoresError(t *testing.T) {
	_, s, _ := analysisFixture(t)
	ctx := context.Background()
	_ = s.Create(ctx, newAnalysis("a1", t0))
	fin := t0.Add(time.Second)
	if err := s.Fail(ctx, "a1", fin, "engine exploded"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(ctx, "a1")
	if got.Status != domain.AnalysisFailed || got.Error != "engine exploded" || got.FinishedAt == nil || !got.FinishedAt.Equal(fin) {
		t.Fatalf("after Fail = %+v", got)
	}
}

func TestGetUnknownIsNotFound(t *testing.T) {
	_, s, _ := analysisFixture(t)
	if _, err := s.Get(context.Background(), "missing"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("Get(missing) = %v, want ErrNotFound", err)
	}
}

func TestLatestAndLatestDone(t *testing.T) {
	_, s, _ := analysisFixture(t)
	ctx := context.Background()

	if a, err := s.Latest(ctx); err != nil || a != nil {
		t.Fatalf("Latest on empty = %v, %v; want nil, nil", a, err)
	}
	if a, err := s.LatestDone(ctx); err != nil || a != nil {
		t.Fatalf("LatestDone on empty = %v, %v; want nil, nil", a, err)
	}

	for i, id := range []string{"old-done", "failed", "running"} {
		if err := s.Create(ctx, newAnalysis(id, t0.Add(time.Duration(i)*time.Minute))); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Complete(ctx, "old-done", t0, domain.Summary{Text: "zero"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Fail(ctx, "failed", t0, "boom"); err != nil {
		t.Fatal(err)
	}
	_ = s.UpdateProgress(ctx, "running", domain.AnalysisRunning, domain.StageBaseline, 2, 14)

	latest, err := s.Latest(ctx)
	if err != nil || latest == nil || latest.ID != "running" {
		t.Fatalf("Latest = %+v, %v; want running", latest, err)
	}
	ld, err := s.LatestDone(ctx)
	if err != nil || ld == nil || ld.ID != "old-done" {
		t.Fatalf("LatestDone = %+v, %v; want old-done (failed and running ignored)", ld, err)
	}

	if err := s.Create(ctx, newAnalysis("new-done", t0.Add(10*time.Minute))); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(ctx, "new-done", t0, domain.Summary{}, nil); err != nil {
		t.Fatal(err)
	}
	if ld, _ = s.LatestDone(ctx); ld == nil || ld.ID != "new-done" {
		t.Fatalf("LatestDone = %+v, want new-done", ld)
	}
}

func TestFailInterrupted(t *testing.T) {
	_, s, _ := analysisFixture(t)
	ctx := context.Background()
	for i, id := range []string{"pend", "run", "done", "failed"} {
		_ = s.Create(ctx, newAnalysis(id, t0.Add(time.Duration(i)*time.Minute)))
	}
	_ = s.UpdateProgress(ctx, "run", domain.AnalysisRunning, domain.StageBaseline, 2, 14)
	_ = s.Complete(ctx, "done", t0, domain.Summary{}, nil)
	_ = s.Fail(ctx, "failed", t0, "earlier failure")

	at := t0.Add(time.Hour)
	n, err := s.FailInterrupted(ctx, at)
	if err != nil || n != 2 {
		t.Fatalf("FailInterrupted = %d, %v; want 2", n, err)
	}
	for _, id := range []string{"pend", "run"} {
		a, _ := s.Get(ctx, id)
		if a.Status != domain.AnalysisFailed || a.Error != "interrupted by restart" || a.FinishedAt == nil || !a.FinishedAt.Equal(at) {
			t.Fatalf("%s after FailInterrupted = %+v", id, a)
		}
	}
	if a, _ := s.Get(ctx, "done"); a.Status != domain.AnalysisDone {
		t.Fatalf("done analysis must stay done: %+v", a)
	}
	if a, _ := s.Get(ctx, "failed"); a.Error != "earlier failure" {
		t.Fatalf("failed analysis keeps its own error: %+v", a)
	}
	if n, _ := s.FailInterrupted(ctx, at); n != 0 {
		t.Fatalf("second FailInterrupted = %d, want 0", n)
	}
}

func TestAnomalyListOrderAndFilters(t *testing.T) {
	_, s, an := analysisFixture(t)
	ctx := context.Background()
	_ = s.Create(ctx, newAnalysis("a1", t0))
	_ = s.Create(ctx, newAnalysis("a2", t0.Add(time.Minute)))
	as := []domain.Anomaly{
		anomaly("r3", "a1", "C-3", 3, domain.TypeExplainable, domain.SeverityMedium),
		anomaly("r1", "a1", "A-1", 1, domain.TypeReal, domain.SeverityHigh),
		anomaly("r2", "a1", "B-2", 2, domain.TypeDataQuality, domain.SeverityHigh),
	}
	if err := s.Complete(ctx, "a1", t0, domain.Summary{Total: 3}, as); err != nil {
		t.Fatal(err)
	}
	other := []domain.Anomaly{anomaly("z1", "a2", "A-1", 1, domain.TypeReal, domain.SeverityHigh)}
	if err := s.Complete(ctx, "a2", t0, domain.Summary{Total: 1}, other); err != nil {
		t.Fatal(err)
	}

	ids := func(f app.AnomalyFilter) []string {
		t.Helper()
		list, err := an.List(ctx, "a1", f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, a := range list {
			out = append(out, a.ID)
		}
		return out
	}
	eq := func(got []string, want ...string) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("got %v, want %v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("got %v, want %v", got, want)
			}
		}
	}
	eq(ids(app.AnomalyFilter{}), "r1", "r2", "r3")
	eq(ids(app.AnomalyFilter{Type: domain.TypeDataQuality}), "r2")
	eq(ids(app.AnomalyFilter{Severity: domain.SeverityHigh}), "r1", "r2")
	eq(ids(app.AnomalyFilter{MeterID: "C-3"}), "r3")
	eq(ids(app.AnomalyFilter{Severity: domain.SeverityHigh, Type: domain.TypeReal}), "r1")
	eq(ids(app.AnomalyFilter{MeterID: "A-1", Type: domain.TypeDataQuality}))
}

func TestAnomalyGetRestoresEvidenceAndEvent(t *testing.T) {
	db, s, an := analysisFixture(t)
	ctx := context.Background()
	var eventID int64
	if err := db.QueryRowContext(ctx, "SELECT id FROM events WHERE meter_id = 'A-1'").Scan(&eventID); err != nil {
		t.Fatal(err)
	}
	_ = s.Create(ctx, newAnalysis("a1", t0))
	withEvent := anomaly("r1", "a1", "A-1", 1, domain.TypeReal, domain.SeverityHigh)
	withEvent.RelatedEvent = &domain.Event{ID: eventID}
	withEvent.Evidence = append(withEvent.Evidence, domain.Evidence{Metric: "event", Label: "Event", Observed: 2, Unit: "h", Detail: "UNKNOWN: firmware update", EventID: &eventID})
	plain := anomaly("r2", "a1", "B-2", 2, domain.TypeDataQuality, domain.SeverityLow)
	if err := s.Complete(ctx, "a1", t0, domain.Summary{Total: 2}, []domain.Anomaly{withEvent, plain}); err != nil {
		t.Fatal(err)
	}

	got, err := an.Get(ctx, "r1")
	if err != nil {
		t.Fatal(err)
	}
	if got.AnalysisID != "a1" || got.MeterID != "A-1" || got.Rank != 1 || got.Kind != domain.KindLevelShiftUp || got.Type != domain.TypeReal ||
		got.Severity != domain.SeverityHigh || got.SeverityScore != 88.5 || got.Confidence != 0.9 || got.PriorityScore != 70.2 ||
		got.Action != domain.ActionInspectLoadIncrease || got.Reason != "reason r1" || got.RecommendedAction != "act r1" || got.Status != "open" {
		t.Fatalf("scalar round trip = %+v", got)
	}
	if !got.WindowStart.Equal(t0) || !got.WindowEnd.Equal(t0.Add(5*time.Hour)) || !got.DetectedAt.Equal(t0) {
		t.Fatalf("times = %v %v %v", got.DetectedAt, got.WindowStart, got.WindowEnd)
	}
	if len(got.Evidence) != 2 || got.Evidence[0].Metric != "consumption_kwh" || *got.Evidence[0].Baseline != 20 || got.Evidence[0].Observed != 42 ||
		*got.Evidence[0].DeltaPct != 110 || got.Evidence[1].EventID == nil || *got.Evidence[1].EventID != eventID || got.Evidence[1].Baseline != nil {
		t.Fatalf("evidence round trip = %+v", got.Evidence)
	}
	if got.RelatedEvent == nil || got.RelatedEvent.ID != eventID || got.RelatedEvent.Type != domain.EventUnknown ||
		got.RelatedEvent.Description != "firmware update" || got.RelatedEvent.MeterID != "A-1" {
		t.Fatalf("related event = %+v", got.RelatedEvent)
	}

	noEv, err := an.Get(ctx, "r2")
	if err != nil || noEv.RelatedEvent != nil {
		t.Fatalf("anomaly without event = %+v, %v", noEv, err)
	}
	if _, err := an.Get(ctx, "missing"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("Get(missing) = %v, want ErrNotFound", err)
	}

	list, _ := an.List(ctx, "a1", app.AnomalyFilter{})
	if len(list) != 2 || list[0].RelatedEvent == nil || list[0].RelatedEvent.Description != "firmware update" || len(list[0].Evidence) != 2 {
		t.Fatalf("List must restore evidence and event: %+v", list)
	}
}
