package httpapi

import (
	"net/http"
	"time"

	"energy/internal/domain"
)

type analyzeDTO struct {
	ID string `json:"id"`
}

type summaryDTO struct {
	Total             int            `json:"total"`
	PriorityAttention int            `json:"priority_attention"`
	MetersAnalyzed    int            `json:"meters_analyzed"`
	ByType            map[string]int `json:"by_type"`
	BySeverity        map[string]int `json:"by_severity"`
	AvgConfidence     float64        `json:"avg_confidence"`
	Text              string         `json:"text"`
}

type analysisDTO struct {
	ID         string      `json:"id"`
	Status     string      `json:"status"`
	Stage      string      `json:"stage"`
	StageIndex int         `json:"stage_index"`
	StageTotal int         `json:"stage_total"`
	Progress   int         `json:"progress"`
	CreatedAt  time.Time   `json:"created_at"`
	StartedAt  *time.Time  `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at"`
	Error      string      `json:"error"`
	Summary    *summaryDTO `json:"summary"`
}

// stageTotal is the number of pipeline stages a run reports.
const stageTotal = 7

func toAnalysisDTO(a domain.Analysis) analysisDTO {
	out := analysisDTO{
		ID: a.ID, Status: string(a.Status), Stage: string(a.Stage), StageIndex: a.StageIndex, StageTotal: stageTotal,
		Progress: a.Progress, CreatedAt: a.CreatedAt, StartedAt: a.StartedAt, FinishedAt: a.FinishedAt, Error: a.Error,
	}
	if a.Status == domain.AnalysisDone && a.Summary != nil {
		s := a.Summary
		sum := summaryDTO{
			Total: s.Total, PriorityAttention: s.PriorityAttention, MetersAnalyzed: s.MetersAnalyzed,
			ByType: map[string]int{}, BySeverity: map[string]int{}, AvgConfidence: s.AvgConfidence, Text: s.Text,
		}
		for k, v := range s.ByType {
			sum.ByType[string(k)] = v
		}
		for k, v := range s.BySeverity {
			sum.BySeverity[string(k)] = v
		}
		out.Summary = &sum
	}
	return out
}

func (s *server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	id, err := s.deps.Analysis.Start(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, analyzeDTO{ID: id})
}

func (s *server) handleGetAnalysis(w http.ResponseWriter, r *http.Request) {
	a, err := s.deps.Analysis.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAnalysisDTO(a))
}

func (s *server) handleLatestAnalysis(w http.ResponseWriter, r *http.Request) {
	a, err := s.deps.Analysis.Latest(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	if a == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, toAnalysisDTO(*a))
}
