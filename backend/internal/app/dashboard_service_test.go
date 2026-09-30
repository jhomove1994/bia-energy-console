package app

import (
	"context"
	"math"
	"testing"
	"time"

	"energy/internal/domain"
)

type staticLatest struct{ a *domain.Analysis }

func (s staticLatest) LatestDone(context.Context) (*domain.Analysis, error) { return s.a, nil }

func TestDashboardBeforeAnyAnalysis(t *testing.T) {
	store := demoStore()
	svc := NewDashboardService(store, store, NoLatestDone{})
	d, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	total, _ := store.TotalKWh(context.Background())
	if d.MetersCount != 3 || math.Abs(d.TotalKWh-total) > 1e-9 || math.Abs(d.TotalKWh-10346.4) > 1e-6 {
		t.Fatalf("meters=%d total=%v (want sum %v)", d.MetersCount, d.TotalKWh, total)
	}
	if d.AnomaliesCount != 0 || d.HighPriorityCount != 0 {
		t.Fatalf("counts = %d/%d, want 0/0", d.AnomaliesCount, d.HighPriorityCount)
	}
	if d.AggregateConfidence != nil || d.LastAnalysis != nil || d.SummaryText != nil {
		t.Fatalf("nullable fields must be nil: %+v", d)
	}
}

func TestDashboardUsesLatestDoneAnalysis(t *testing.T) {
	finished := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		summary domain.Summary
		wantAgg *float64
	}{
		{"with anomalies", domain.Summary{Total: 4, PriorityAttention: 2, AvgConfidence: 0.75, Text: "4 anomalies detected · 2 require priority attention"}, ptr(0.75)},
		{"clean run has no confidence", domain.Summary{Total: 0, Text: "0 anomalies detected · 0 require priority attention"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := demoStore()
			a := &domain.Analysis{ID: "an-1", Status: domain.AnalysisDone, FinishedAt: &finished, Summary: &tt.summary}
			d, err := NewDashboardService(store, store, staticLatest{a}).Summary(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if d.AnomaliesCount != tt.summary.Total || d.HighPriorityCount != tt.summary.PriorityAttention {
				t.Fatalf("counts = %d/%d", d.AnomaliesCount, d.HighPriorityCount)
			}
			if (d.AggregateConfidence == nil) != (tt.wantAgg == nil) || (tt.wantAgg != nil && *d.AggregateConfidence != *tt.wantAgg) {
				t.Fatalf("aggregate = %v, want %v", d.AggregateConfidence, tt.wantAgg)
			}
			if d.SummaryText == nil || *d.SummaryText != tt.summary.Text {
				t.Fatalf("summary text = %v", d.SummaryText)
			}
			if d.LastAnalysis == nil || d.LastAnalysis.ID != "an-1" || d.LastAnalysis.Status != "done" || !d.LastAnalysis.FinishedAt.Equal(finished) {
				t.Fatalf("last analysis = %+v", d.LastAnalysis)
			}
		})
	}
}

func ptr(v float64) *float64 { return &v }
