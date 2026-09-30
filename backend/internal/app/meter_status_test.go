package app

import (
	"context"
	"errors"
	"testing"

	"energy/internal/domain"
)

func meterAn(meter string, t domain.AnomalyType, s domain.Severity) domain.Anomaly {
	return domain.Anomaly{MeterID: meter, Type: t, Severity: s}
}

func TestDeriveStatuses(t *testing.T) {
	tests := []struct {
		name  string
		in    []domain.Anomaly
		want  MeterStatus
		count int
	}{
		{"real high is critical", []domain.Anomaly{meterAn("A", domain.TypeReal, domain.SeverityHigh)}, StatusCritical, 1},
		{"real medium is alert", []domain.Anomaly{meterAn("A", domain.TypeReal, domain.SeverityMedium)}, StatusAlert, 1},
		{"real low is alert", []domain.Anomaly{meterAn("A", domain.TypeReal, domain.SeverityLow)}, StatusAlert, 1},
		{"data quality high is only alert", []domain.Anomaly{meterAn("A", domain.TypeDataQuality, domain.SeverityHigh)}, StatusAlert, 1},
		{"explainable high is only alert", []domain.Anomaly{meterAn("A", domain.TypeExplainable, domain.SeverityHigh)}, StatusAlert, 1},
		{"false positive only stays ok", []domain.Anomaly{meterAn("A", domain.TypeFalsePos, domain.SeverityHigh)}, StatusOK, 1},
		{"false positive plus explainable is alert", []domain.Anomaly{
			meterAn("A", domain.TypeFalsePos, domain.SeverityLow), meterAn("A", domain.TypeExplainable, domain.SeverityMedium)}, StatusAlert, 2},
		{"critical wins over alert", []domain.Anomaly{
			meterAn("A", domain.TypeDataQuality, domain.SeverityHigh), meterAn("A", domain.TypeReal, domain.SeverityHigh)}, StatusCritical, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveStatuses(tt.in)["A"]
			if got.Status != tt.want || got.AnomalyCount != tt.count {
				t.Fatalf("finding = %+v, want %s x%d", got, tt.want, tt.count)
			}
		})
	}
}

func TestDeriveStatusesGroupsPerMeter(t *testing.T) {
	got := DeriveStatuses([]domain.Anomaly{
		meterAn("A", domain.TypeReal, domain.SeverityHigh), meterAn("B", domain.TypeDataQuality, domain.SeverityHigh),
		meterAn("C", domain.TypeFalsePos, domain.SeverityLow)})
	if len(got) != 3 || got["A"].Status != StatusCritical || got["B"].Status != StatusAlert || got["C"].Status != StatusOK {
		t.Fatalf("statuses = %+v", got)
	}
	if _, ok := got["D"]; ok || len(DeriveStatuses(nil)) != 0 {
		t.Fatal("meters without anomalies are absent (OK by default)")
	}
}

type fakeAnomalies struct {
	byAnalysis map[string][]domain.Anomaly
	askedFor   []string
	err        error
}

func (f *fakeAnomalies) List(_ context.Context, id string, _ AnomalyFilter) ([]domain.Anomaly, error) {
	f.askedFor = append(f.askedFor, id)
	return f.byAnalysis[id], f.err
}

func (f *fakeAnomalies) Get(context.Context, string) (domain.Anomaly, error) {
	return domain.Anomaly{}, ErrNotFound
}

func TestAnalysisStatusesUsesLatestDoneOnly(t *testing.T) {
	ctx := context.Background()
	anoms := &fakeAnomalies{byAnalysis: map[string][]domain.Anomaly{
		"old":    {meterAn("M-1", domain.TypeExplainable, domain.SeverityMedium)},
		"latest": {meterAn("M-1", domain.TypeReal, domain.SeverityHigh), meterAn("M-2", domain.TypeDataQuality, domain.SeverityHigh)},
	}}

	// No analysis yet: every meter is OK (empty map).
	src := NewAnalysisStatuses(staticLatest{nil}, anoms)
	got, err := src.Statuses(ctx)
	if err != nil || len(got) != 0 || len(anoms.askedFor) != 0 {
		t.Fatalf("no analysis: %v, %v, asked %v", got, err, anoms.askedFor)
	}

	// The done analysis feeds the statuses, through the same interface the meter service uses.
	src = NewAnalysisStatuses(staticLatest{&domain.Analysis{ID: "latest", Status: domain.AnalysisDone}}, anoms)
	svc := newService(demoStore(), src)
	rows, err := svc.List(ctx, ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	status := map[string]MeterStatus{}
	for _, r := range rows {
		status[r.MeterID] = r.Status
	}
	// demoStore has M-101, M-109, M-110; M-1 and M-2 are unknown meters and must not leak in.
	if status["M-101"] != StatusOK || status["M-109"] != StatusOK || len(rows) != 3 {
		t.Fatalf("rows = %+v", status)
	}
	f, err := src.Statuses(ctx)
	if err != nil || f["M-1"].Status != StatusCritical || f["M-2"].Status != StatusAlert || anoms.askedFor[len(anoms.askedFor)-1] != "latest" {
		t.Fatalf("statuses = %+v, %v, asked %v", f, err, anoms.askedFor)
	}

	anoms.err = errors.New("db down")
	if _, err := src.Statuses(ctx); err == nil {
		t.Fatal("a repository error must surface")
	}
}

func TestMeterListShowsDerivedStatusAndCount(t *testing.T) {
	store := demoStore()
	anoms := &fakeAnomalies{byAnalysis: map[string][]domain.Anomaly{"an": {
		meterAn("M-109", domain.TypeReal, domain.SeverityHigh),
		meterAn("M-110", domain.TypeFalsePos, domain.SeverityLow),
	}}}
	src := NewAnalysisStatuses(staticLatest{&domain.Analysis{ID: "an", Status: domain.AnalysisDone}}, anoms)
	svc := newService(store, src)

	rows, err := svc.List(context.Background(), ListQuery{})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]MeterStatus{}
	for _, r := range rows {
		got[r.MeterID] = r.Status
	}
	if got["M-109"] != StatusCritical || got["M-110"] != StatusOK || got["M-101"] != StatusOK {
		t.Fatalf("statuses = %v", got)
	}
	d, err := svc.Detail(context.Background(), "M-109")
	if err != nil || d.Status != StatusCritical || d.FindingsCount != 1 {
		t.Fatalf("detail = %+v, %v", d, err)
	}
	crit, err := svc.List(context.Background(), ListQuery{Status: "critical"})
	if err != nil || len(crit) != 1 || crit[0].MeterID != "M-109" {
		t.Fatalf("critical filter = %+v, %v", crit, err)
	}
}
