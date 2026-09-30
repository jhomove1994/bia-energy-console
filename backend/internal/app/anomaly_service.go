package app

import (
	"context"

	"energy/internal/domain"
)

// AnomalyQuery selects anomalies. AnalysisID defaults to the latest done
// analysis; the other fields are optional filters.
type AnomalyQuery struct {
	AnalysisID string
	MeterID    string
	Type       string
	Severity   string
}

// AnomalyList is the anomalies of one analysis in rank order. AnalysisID and
// SummaryText are empty/nil when no analysis has completed.
type AnomalyList struct {
	AnalysisID  string
	SummaryText *string
	Items       []domain.Anomaly
}

// AnomalyService serves the anomaly list and detail.
type AnomalyService struct {
	analyses  AnalysisRepository
	anomalies AnomalyRepository
}

// NewAnomalyService wires the service to its ports.
func NewAnomalyService(a AnalysisRepository, an AnomalyRepository) *AnomalyService {
	return &AnomalyService{analyses: a, anomalies: an}
}

// List returns the anomalies of the requested analysis, or of the latest done
// one. An unknown analysis id is ErrNotFound. Only done analyses expose
// anomalies, so a failed or running run yields an empty list.
func (s *AnomalyService) List(ctx context.Context, q AnomalyQuery) (AnomalyList, error) {
	filter, err := parseFilter(q)
	if err != nil {
		return AnomalyList{}, err
	}
	var a *domain.Analysis
	if q.AnalysisID != "" {
		got, err := s.analyses.Get(ctx, q.AnalysisID)
		if err != nil {
			return AnomalyList{}, err
		}
		a = &got
	} else if a, err = s.analyses.LatestDone(ctx); err != nil {
		return AnomalyList{}, err
	}
	out := AnomalyList{Items: []domain.Anomaly{}}
	if a == nil {
		return out, nil
	}
	out.AnalysisID = a.ID
	if a.Status != domain.AnalysisDone {
		return out, nil
	}
	if a.Summary != nil {
		text := a.Summary.Text
		out.SummaryText = &text
	}
	items, err := s.anomalies.List(ctx, a.ID, filter)
	if err != nil {
		return AnomalyList{}, err
	}
	if items != nil {
		out.Items = items
	}
	return out, nil
}

// Get returns one anomaly with evidence and related event; ErrNotFound if unknown.
func (s *AnomalyService) Get(ctx context.Context, id string) (domain.Anomaly, error) {
	return s.anomalies.Get(ctx, id)
}

func parseFilter(q AnomalyQuery) (AnomalyFilter, error) {
	f := AnomalyFilter{MeterID: q.MeterID}
	if q.Type != "" {
		switch t := domain.AnomalyType(q.Type); t {
		case domain.TypeReal, domain.TypeExplainable, domain.TypeFalsePos, domain.TypeDataQuality:
			f.Type = t
		default:
			return f, invalid("invalid type %q: expected REAL_ANOMALY, EXPLAINABLE_ANOMALY, FALSE_POSITIVE or DATA_QUALITY", q.Type)
		}
	}
	if q.Severity != "" {
		switch s := domain.Severity(q.Severity); s {
		case domain.SeverityHigh, domain.SeverityMedium, domain.SeverityLow:
			f.Severity = s
		default:
			return f, invalid("invalid severity %q: expected HIGH, MEDIUM or LOW", q.Severity)
		}
	}
	return f, nil
}
