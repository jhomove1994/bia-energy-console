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
