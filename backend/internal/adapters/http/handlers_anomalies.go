package httpapi

import (
	"net/http"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
	"energy/internal/engine"
)

type evidenceDTO struct {
	Metric   string   `json:"metric"`
	Label    string   `json:"label"`
	Baseline *float64 `json:"baseline"`
	Observed float64  `json:"observed"`
	DeltaPct *float64 `json:"delta_pct"`
	Unit     string   `json:"unit"`
	Detail   string   `json:"detail"`
}

type anomalyDTO struct {
	ID                string        `json:"id"`
	AnalysisID        string        `json:"analysis_id"`
	MeterID           string        `json:"meter_id"`
	Anomaly           string        `json:"anomaly"`
	Kind              string        `json:"kind"`
	Type              string        `json:"type"`
	Severity          string        `json:"severity"`
	SeverityScore     float64       `json:"severity_score"`
	Confidence        float64       `json:"confidence"`
	ConfidenceLabel   string        `json:"confidence_label"`
	PriorityScore     float64       `json:"priority_score"`
	PriorityRank      int           `json:"priority_rank"`
	DetectedAt        time.Time     `json:"detected_at"`
	WindowStart       time.Time     `json:"window_start"`
	WindowEnd         time.Time     `json:"window_end"`
	Reason            string        `json:"reason"`
	RecommendedAction string        `json:"recommended_action"`
	ActionCode        string        `json:"action_code"`
	Status            string        `json:"status"`
	RelatedEvent      *eventDTO     `json:"related_event"`
	Evidence          []evidenceDTO `json:"evidence"`
}

type anomalyListDTO struct {
	AnalysisID  *string      `json:"analysis_id"`
	SummaryText *string      `json:"summary_text"`
	Items       []anomalyDTO `json:"items"`
}

// kindLabels are the short human titles of the anomaly kinds.
var kindLabels = map[domain.AnomalyKind]string{
	domain.KindConsumptionSpike:  "Consumption spike",
	domain.KindConsumptionDip:    "Consumption dip",
	domain.KindSustainedIncrease: "Sustained increase",
	domain.KindSustainedDecrease: "Sustained decrease",
	domain.KindLevelShiftUp:      "Level shift up",
	domain.KindLevelShiftDown:    "Level shift down",
	domain.KindDataInconsistency: "Data inconsistency",
	domain.KindPatternChange:     "Pattern change",
}

func kindLabel(k domain.AnomalyKind) string {
	if l, ok := kindLabels[k]; ok {
		return l
	}
	return string(k)
}

func toAnomalyDTO(a domain.Anomaly, cfg engine.Config) anomalyDTO {
	out := anomalyDTO{
		ID: a.ID, AnalysisID: a.AnalysisID, MeterID: a.MeterID, Anomaly: kindLabel(a.Kind), Kind: string(a.Kind),
		Type: string(a.Type), Severity: string(a.Severity), SeverityScore: a.SeverityScore, Confidence: a.Confidence,
		ConfidenceLabel: engine.ConfidenceLabel(a.Confidence, cfg), PriorityScore: a.PriorityScore, PriorityRank: a.Rank,
		DetectedAt: a.DetectedAt, WindowStart: a.WindowStart, WindowEnd: a.WindowEnd, Reason: a.Reason,
		RecommendedAction: a.RecommendedAction, ActionCode: string(a.Action), Status: a.Status,
		Evidence: make([]evidenceDTO, len(a.Evidence)),
	}
	if a.RelatedEvent != nil {
		ev := toEventDTO(*a.RelatedEvent)
		out.RelatedEvent = &ev
	}
	for i, e := range a.Evidence {
		out.Evidence[i] = evidenceDTO{Metric: e.Metric, Label: e.Label, Baseline: e.Baseline, Observed: e.Observed, DeltaPct: e.DeltaPct, Unit: e.Unit, Detail: e.Detail}
	}
	return out
}

func (s *server) handleListAnomalies(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()
	list, err := s.deps.Anomalies.List(r.Context(), app.AnomalyQuery{
		AnalysisID: qp.Get("analysis_id"), MeterID: qp.Get("meter_id"), Type: qp.Get("type"), Severity: qp.Get("severity"),
	})
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	out := anomalyListDTO{SummaryText: list.SummaryText, Items: make([]anomalyDTO, len(list.Items))}
	if list.AnalysisID != "" {
		id := list.AnalysisID
		out.AnalysisID = &id
	}
	for i, a := range list.Items {
		out.Items[i] = toAnomalyDTO(a, s.cfg)
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *server) handleGetAnomaly(w http.ResponseWriter, r *http.Request) {
	a, err := s.deps.Anomalies.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAnomalyDTO(a, s.cfg))
}
