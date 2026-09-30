package app

import (
	"fmt"
	"math"

	"energy/internal/domain"
)

// Summarize aggregates the anomalies of one analysis. Total counts every
// anomaly, false positives included; PriorityAttention counts HIGH severity
// anomalies that are REAL_ANOMALY or DATA_QUALITY. AvgConfidence is rounded to
// three decimals.
func Summarize(as []domain.Anomaly, metersAnalyzed int) domain.Summary {
	s := domain.Summary{
		Total:          len(as),
		MetersAnalyzed: metersAnalyzed,
		ByType:         map[domain.AnomalyType]int{},
		BySeverity:     map[domain.Severity]int{},
	}
	var conf float64
	for _, a := range as {
		s.ByType[a.Type]++
		s.BySeverity[a.Severity]++
		conf += a.Confidence
		if a.Severity == domain.SeverityHigh && (a.Type == domain.TypeReal || a.Type == domain.TypeDataQuality) {
			s.PriorityAttention++
		}
	}
	if s.Total > 0 {
		s.AvgConfidence = math.Round(conf/float64(s.Total)*1000) / 1000
	}
	s.Text = summaryText(s.Total, s.PriorityAttention)
	return s
}

// summaryText renders the headline with U+00B7 and singular forms for 1.
func summaryText(total, priority int) string {
	found := fmt.Sprintf("%d anomalías detectadas", total)
	if total == 1 {
		found = "1 anomalía detectada"
	}
	need := fmt.Sprintf("%d requieren atención prioritaria", priority)
	if priority == 1 {
		need = "1 requiere atención prioritaria"
	}
	return found + " · " + need
}
