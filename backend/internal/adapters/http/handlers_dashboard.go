package httpapi

import (
	"net/http"
	"time"
)

type lastAnalysisDTO struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	FinishedAt *time.Time `json:"finished_at"`
}

type dashboardDTO struct {
	MetersCount         int              `json:"meters_count"`
	TotalKWh            float64          `json:"total_kwh"`
	AnomaliesCount      int              `json:"anomalies_count"`
	HighPriorityCount   int              `json:"high_priority_count"`
	AggregateConfidence *float64         `json:"aggregate_confidence"`
	LastAnalysis        *lastAnalysisDTO `json:"last_analysis"`
	SummaryText         *string          `json:"summary_text"`
}

func (s *server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	d, err := s.deps.Dashboard.Summary(r.Context())
	if err != nil {
		s.writeServiceError(w, r, err)
		return
	}
	out := dashboardDTO{
		MetersCount: d.MetersCount, TotalKWh: d.TotalKWh, AnomaliesCount: d.AnomaliesCount,
		HighPriorityCount: d.HighPriorityCount, AggregateConfidence: d.AggregateConfidence, SummaryText: d.SummaryText,
	}
	if d.LastAnalysis != nil {
		out.LastAnalysis = &lastAnalysisDTO{ID: d.LastAnalysis.ID, Status: d.LastAnalysis.Status, FinishedAt: d.LastAnalysis.FinishedAt}
	}
	writeJSON(w, http.StatusOK, out)
}
