package app

import (
	"math"
	"testing"

	"energy/internal/domain"
)

func an(t domain.AnomalyType, s domain.Severity, conf float64) domain.Anomaly {
	return domain.Anomaly{Type: t, Severity: s, Confidence: conf}
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name     string
		in       []domain.Anomaly
		total    int
		priority int
		text     string
	}{
		{"demo shape", []domain.Anomaly{
			an(domain.TypeReal, domain.SeverityHigh, 0.975), an(domain.TypeDataQuality, domain.SeverityHigh, 0.98),
			an(domain.TypeExplainable, domain.SeverityMedium, 0.84), an(domain.TypeFalsePos, domain.SeverityLow, 0.86)},
			4, 2, "4 anomalies detected · 2 require priority attention"},
		{"none", nil, 0, 0, "0 anomalies detected · 0 require priority attention"},
		{"one priority", []domain.Anomaly{an(domain.TypeReal, domain.SeverityHigh, 0.9)}, 1, 1, "1 anomaly detected · 1 requires priority attention"},
		{"one without priority", []domain.Anomaly{an(domain.TypeExplainable, domain.SeverityHigh, 0.9)}, 1, 0, "1 anomaly detected · 0 require priority attention"},
		{"false positive counts in total only", []domain.Anomaly{an(domain.TypeFalsePos, domain.SeverityHigh, 0.9), an(domain.TypeReal, domain.SeverityMedium, 0.8)},
			2, 0, "2 anomalies detected · 0 require priority attention"},
		{"high explainable is not priority", []domain.Anomaly{an(domain.TypeExplainable, domain.SeverityHigh, 0.9), an(domain.TypeDataQuality, domain.SeverityLow, 0.9)},
			2, 0, "2 anomalies detected · 0 require priority attention"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Summarize(tt.in, 12)
			if s.Total != tt.total || s.PriorityAttention != tt.priority || s.Text != tt.text {
				t.Fatalf("Summarize = %d/%d %q, want %d/%d %q", s.Total, s.PriorityAttention, s.Text, tt.total, tt.priority, tt.text)
			}
			if s.MetersAnalyzed != 12 {
				t.Fatalf("MetersAnalyzed = %d, want 12", s.MetersAnalyzed)
			}
		})
	}
}

func TestSummarizeBreakdownsAndConfidence(t *testing.T) {
	s := Summarize([]domain.Anomaly{
		an(domain.TypeReal, domain.SeverityHigh, 0.9), an(domain.TypeReal, domain.SeverityMedium, 0.8),
		an(domain.TypeDataQuality, domain.SeverityHigh, 0.7), an(domain.TypeFalsePos, domain.SeverityLow, 0.6)}, 3)
	if s.ByType[domain.TypeReal] != 2 || s.ByType[domain.TypeDataQuality] != 1 || s.ByType[domain.TypeFalsePos] != 1 || s.ByType[domain.TypeExplainable] != 0 {
		t.Fatalf("ByType = %v", s.ByType)
	}
	if s.BySeverity[domain.SeverityHigh] != 2 || s.BySeverity[domain.SeverityMedium] != 1 || s.BySeverity[domain.SeverityLow] != 1 {
		t.Fatalf("BySeverity = %v", s.BySeverity)
	}
	if math.Abs(s.AvgConfidence-0.75) > 1e-9 {
		t.Fatalf("AvgConfidence = %v, want 0.75", s.AvgConfidence)
	}
	if empty := Summarize(nil, 0); empty.AvgConfidence != 0 || empty.ByType == nil || empty.BySeverity == nil {
		t.Fatalf("empty summary = %+v", empty)
	}
}
