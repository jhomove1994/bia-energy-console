package app

import (
	"context"
	"time"

	"energy/internal/domain"
)

// LatestDoneSource returns the most recent completed analysis, or nil when
// none exists. Failed and running analyses are never returned.
type LatestDoneSource interface {
	LatestDone(ctx context.Context) (*domain.Analysis, error)
}

// NoLatestDone is the default source while no analysis storage exists.
type NoLatestDone struct{}

// LatestDone implements LatestDoneSource.
func (NoLatestDone) LatestDone(context.Context) (*domain.Analysis, error) { return nil, nil }

// LastAnalysis identifies the analysis behind the dashboard figures.
type LastAnalysis struct {
	ID         string
	Status     string
	FinishedAt *time.Time
}

// Dashboard is the KPI payload of the summary endpoint.
type Dashboard struct {
	MetersCount         int
	TotalKWh            float64
	AnomaliesCount      int
	HighPriorityCount   int
	AggregateConfidence *float64
	LastAnalysis        *LastAnalysis
	SummaryText         *string
}

// DashboardService computes dashboard KPIs.
type DashboardService struct {
	meters   MeterRepository
	readings ReadingRepository
	latest   LatestDoneSource
}

// NewDashboardService wires the service to its ports.
func NewDashboardService(m MeterRepository, r ReadingRepository, latest LatestDoneSource) *DashboardService {
	return &DashboardService{meters: m, readings: r, latest: latest}
}

// Summary returns the KPIs. Anomaly figures come from the latest done
// analysis; without one they are zero and the nullable fields are nil.
func (s *DashboardService) Summary(ctx context.Context) (Dashboard, error) {
	meters, err := s.meters.List(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	total, err := s.readings.TotalKWh(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	d := Dashboard{MetersCount: len(meters), TotalKWh: total}

	a, err := s.latest.LatestDone(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	if a == nil {
		return d, nil
	}
	d.LastAnalysis = &LastAnalysis{ID: a.ID, Status: string(a.Status), FinishedAt: a.FinishedAt}
	if sum := a.Summary; sum != nil {
		d.AnomaliesCount = sum.Total
		d.HighPriorityCount = sum.PriorityAttention
		text := sum.Text
		d.SummaryText = &text
		if sum.Total > 0 {
			conf := sum.AvgConfidence
			d.AggregateConfidence = &conf
		}
	}
	return d, nil
}
