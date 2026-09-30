package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"energy/internal/domain"
)

// filteringAnomalies applies the filter like the SQL adapter does.
type filteringAnomalies struct{ fakeAnomalies }

func (f *filteringAnomalies) List(ctx context.Context, id string, flt AnomalyFilter) ([]domain.Anomaly, error) {
	all, _ := f.fakeAnomalies.List(ctx, id, flt)
	var out []domain.Anomaly
	for _, a := range all {
		if (flt.MeterID == "" || a.MeterID == flt.MeterID) && (flt.Type == "" || a.Type == flt.Type) && (flt.Severity == "" || a.Severity == flt.Severity) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *filteringAnomalies) Get(_ context.Context, id string) (domain.Anomaly, error) {
	for _, list := range f.byAnalysis {
		for _, a := range list {
			if a.ID == id {
				return a, nil
			}
		}
	}
	return domain.Anomaly{}, ErrNotFound
}

func anomalyServiceFixture(t *testing.T) (*AnomalyService, *fakeAnalyses) {
	t.Helper()
	ctx := context.Background()
	repo := newFakeAnalyses()
	for _, id := range []string{"old", "new", "broken"} {
		_ = repo.Create(ctx, domain.Analysis{ID: id, Status: domain.AnalysisPending})
	}
	_ = repo.Complete(ctx, "old", time.Now(), domain.Summary{Text: "old text"}, nil)
	_ = repo.Complete(ctx, "new", time.Now(), domain.Summary{Total: 2, Text: "new text"}, nil)
	_ = repo.Fail(ctx, "broken", time.Now(), "boom")
	anoms := &filteringAnomalies{fakeAnomalies{byAnalysis: map[string][]domain.Anomaly{
		"old": {{ID: "o1", MeterID: "M-1", Type: domain.TypeReal, Severity: domain.SeverityHigh}},
		"new": {
			{ID: "n1", MeterID: "M-1", Type: domain.TypeReal, Severity: domain.SeverityHigh},
			{ID: "n2", MeterID: "M-2", Type: domain.TypeDataQuality, Severity: domain.SeverityMedium},
		},
	}}}
	return NewAnomalyService(repo, anoms), repo
}

func TestAnomalyListDefaultsToLatestDone(t *testing.T) {
	svc, _ := anomalyServiceFixture(t)
	got, err := svc.List(context.Background(), AnomalyQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got.AnalysisID != "new" || got.SummaryText == nil || *got.SummaryText != "new text" || len(got.Items) != 2 || got.Items[0].ID != "n1" {
		t.Fatalf("list = %+v (failed newest run must be ignored)", got)
	}
}

func TestAnomalyListBeforeAnyDoneAnalysis(t *testing.T) {
	repo := newFakeAnalyses()
	_ = repo.Create(context.Background(), domain.Analysis{ID: "p", Status: domain.AnalysisPending})
	got, err := NewAnomalyService(repo, &filteringAnomalies{}).List(context.Background(), AnomalyQuery{})
	if err != nil || got.AnalysisID != "" || got.SummaryText != nil || got.Items == nil || len(got.Items) != 0 {
		t.Fatalf("list = %+v, %v; want empty non-nil items", got, err)
	}
}

func TestAnomalyListExplicitAnalysisAndFilters(t *testing.T) {
	svc, _ := anomalyServiceFixture(t)
	ctx := context.Background()

	old, err := svc.List(ctx, AnomalyQuery{AnalysisID: "old"})
	if err != nil || old.AnalysisID != "old" || len(old.Items) != 1 || *old.SummaryText != "old text" {
		t.Fatalf("old = %+v, %v", old, err)
	}
	dq, err := svc.List(ctx, AnomalyQuery{Type: "DATA_QUALITY"})
	if err != nil || len(dq.Items) != 1 || dq.Items[0].ID != "n2" {
		t.Fatalf("type filter = %+v, %v", dq, err)
	}
	sev, _ := svc.List(ctx, AnomalyQuery{Severity: "HIGH", MeterID: "M-1"})
	if len(sev.Items) != 1 || sev.Items[0].ID != "n1" {
		t.Fatalf("severity+meter filter = %+v", sev)
	}
	none, err := svc.List(ctx, AnomalyQuery{MeterID: "M-404"})
	if err != nil || len(none.Items) != 0 {
		t.Fatalf("meter with no findings = %+v, %v", none, err)
	}
	failed, err := svc.List(ctx, AnomalyQuery{AnalysisID: "broken"})
	if err != nil || len(failed.Items) != 0 || failed.SummaryText != nil {
		t.Fatalf("failed analysis exposes nothing: %+v, %v", failed, err)
	}
}

func TestAnomalyListErrors(t *testing.T) {
	svc, _ := anomalyServiceFixture(t)
	ctx := context.Background()
	if _, err := svc.List(ctx, AnomalyQuery{AnalysisID: "ghost"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown analysis = %v, want ErrNotFound", err)
	}
	for _, q := range []AnomalyQuery{{Type: "NOPE"}, {Severity: "EXTREME"}} {
		var ve *ValidationError
		if _, err := svc.List(ctx, q); !errors.As(err, &ve) {
			t.Fatalf("query %+v = %v, want ValidationError", q, err)
		}
	}
}

func TestAnomalyGet(t *testing.T) {
	svc, _ := anomalyServiceFixture(t)
	a, err := svc.Get(context.Background(), "n2")
	if err != nil || a.MeterID != "M-2" {
		t.Fatalf("Get(n2) = %+v, %v", a, err)
	}
	if _, err := svc.Get(context.Background(), "ghost"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(ghost) = %v", err)
	}
}
