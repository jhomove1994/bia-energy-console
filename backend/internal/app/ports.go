// Package app holds the use cases and the ports they depend on. Adapters
// implement the ports; nothing here imports an adapter.
package app

import (
	"context"
	"errors"
	"time"

	"energy/internal/domain"
)

// ErrNotFound reports that a requested entity does not exist.
var ErrNotFound = errors.New("not found")

// MeterRepository reads the meter catalog.
type MeterRepository interface {
	List(ctx context.Context) ([]domain.Meter, error)
	Exists(ctx context.Context, meterID string) (bool, error)
}

// ReadingRepository reads hourly readings.
type ReadingRepository interface {
	// ByMeter returns the readings of one meter in ascending timestamp order.
	// from and to are inclusive bounds and may be nil.
	ByMeter(ctx context.Context, meterID string, from, to *time.Time) ([]domain.Reading, error)
	// AllByMeter returns every series keyed by meter id, sorted by timestamp.
	AllByMeter(ctx context.Context) (map[string][]domain.Reading, error)
	// TotalKWh is the sum of consumption over all readings.
	TotalKWh(ctx context.Context) (float64, error)
}

// EventRepository reads operational events.
type EventRepository interface {
	All(ctx context.Context) ([]domain.Event, error)
	ByMeter(ctx context.Context, meterID string) ([]domain.Event, error)
}

// AnalysisRepository persists analysis runs and their anomalies.
type AnalysisRepository interface {
	Create(ctx context.Context, a domain.Analysis) error
	UpdateProgress(ctx context.Context, id string, st domain.AnalysisStatus, stage domain.Stage, idx, progress int) error
	// Complete inserts the anomalies and marks the analysis done with progress
	// 100 in one transaction; on failure nothing is visible.
	Complete(ctx context.Context, id string, finished time.Time, s domain.Summary, as []domain.Anomaly) error
	Fail(ctx context.Context, id string, finished time.Time, msg string) error
	// Get returns ErrNotFound for an unknown id.
	Get(ctx context.Context, id string) (domain.Analysis, error)
	// Latest is the most recently created analysis in any state, or nil.
	Latest(ctx context.Context) (*domain.Analysis, error)
	// LatestDone is the most recent done analysis, or nil.
	LatestDone(ctx context.Context) (*domain.Analysis, error)
	// FailInterrupted marks pending and running analyses failed and returns how many.
	FailInterrupted(ctx context.Context, finished time.Time) (int, error)
}

// AnomalyFilter narrows an anomaly list; empty fields match everything.
type AnomalyFilter struct {
	MeterID  string
	Type     domain.AnomalyType
	Severity domain.Severity
}

// AnomalyRepository reads persisted anomalies.
type AnomalyRepository interface {
	// List returns the anomalies of one analysis ordered by rank.
	List(ctx context.Context, analysisID string, f AnomalyFilter) ([]domain.Anomaly, error)
	// Get returns one anomaly with evidence and related event; ErrNotFound if unknown.
	Get(ctx context.Context, id string) (domain.Anomaly, error)
}

// IDGenerator produces unique identifiers.
type IDGenerator interface{ NewID() string }

// Clock provides the current time so services stay deterministic in tests.
type Clock interface {
	Now() time.Time
}

// MeterStatus is the derived health status of a meter.
type MeterStatus string

const (
	StatusOK       MeterStatus = "OK"
	StatusAlert    MeterStatus = "Alert"
	StatusCritical MeterStatus = "Critical"
)

// MeterFinding is the analysis outcome for one meter.
type MeterFinding struct {
	Status       MeterStatus
	AnomalyCount int
}

// MeterStatusSource supplies the derived status and finding count per meter.
// Meters missing from the result are OK with no findings.
type MeterStatusSource interface {
	Statuses(ctx context.Context) (map[string]MeterFinding, error)
}

// NoAnalysis is the MeterStatusSource used while no analysis exists: every
// meter is OK.
type NoAnalysis struct{}

// Statuses implements MeterStatusSource.
func (NoAnalysis) Statuses(context.Context) (map[string]MeterFinding, error) {
	return map[string]MeterFinding{}, nil
}

// Explainer renders the human-readable text of an anomaly from its evidence.
// Implementations must be deterministic and must not need the network.
type Explainer interface {
	// Reason explains why the anomaly was reported.
	Reason(a domain.Anomaly) string
	// Action says what to do about it.
	Action(a domain.Anomaly) string
}
