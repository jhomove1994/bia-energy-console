package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"energy/internal/domain"
	"energy/internal/engine"
)

// ValidationError reports an invalid caller-supplied parameter.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// MeterRow is the catalog view of one meter.
type MeterRow struct {
	MeterID       string
	LastReadingAt *time.Time
	TotalKWh      float64
	// VariationPct is the latest full day's kWh versus the baseline daily
	// total, in percent; nil when it cannot be computed.
	VariationPct *float64
	Status       MeterStatus
}

// MeterDetail is a MeterRow plus its open findings count.
type MeterDetail struct {
	MeterRow
	FindingsCount int
}

// ReadingPoint is one reading with the hour-of-day baseline that applies to it.
type ReadingPoint struct {
	domain.Reading
	Baseline engine.Expected
}

// Sort keys and orders accepted by List.
const (
	SortMeterID   = "meter_id"
	SortVariation = "variation"
	SortKWh       = "kwh"
	SortStatus    = "status"
	OrderAsc      = "asc"
	OrderDesc     = "desc"
)

// ListQuery filters and orders the meter list. Zero values mean "no filter",
// meter_id ascending.
type ListQuery struct {
	Q      string // case-insensitive meter id substring
	Status string // OK, Alert or Critical (case-insensitive)
	Sort   string
	Order  string
}

// DeriveStatuses turns the anomalies of one analysis into per-meter findings.
// Critical means a HIGH REAL_ANOMALY; Alert means any DATA_QUALITY or
// EXPLAINABLE_ANOMALY, or a REAL_ANOMALY that is not HIGH; a meter with only
// FALSE_POSITIVE anomalies stays OK. Meters without anomalies are absent.
func DeriveStatuses(as []domain.Anomaly) map[string]MeterFinding {
	out := map[string]MeterFinding{}
	for _, a := range as {
		f := out[a.MeterID]
		f.AnomalyCount++
		st := anomalyStatus(a)
		if statusRank(st) > statusRank(f.Status) || f.Status == "" {
			f.Status = st
		}
		out[a.MeterID] = f
	}
	return out
}

func anomalyStatus(a domain.Anomaly) MeterStatus {
	switch {
	case a.Type == domain.TypeReal && a.Severity == domain.SeverityHigh:
		return StatusCritical
	case a.Type == domain.TypeFalsePos:
		return StatusOK
	default:
		return StatusAlert
	}
}

// AnalysisStatuses is the MeterStatusSource backed by the latest done analysis.
type AnalysisStatuses struct {
	latest    LatestDoneSource
	anomalies AnomalyRepository
}

// NewAnalysisStatuses derives meter statuses from the latest done analysis.
func NewAnalysisStatuses(latest LatestDoneSource, anomalies AnomalyRepository) *AnalysisStatuses {
	return &AnalysisStatuses{latest: latest, anomalies: anomalies}
}

// Statuses implements MeterStatusSource. Without a done analysis every meter is
// OK, so the map is empty.
func (s *AnalysisStatuses) Statuses(ctx context.Context) (map[string]MeterFinding, error) {
	a, err := s.latest.LatestDone(ctx)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return map[string]MeterFinding{}, nil
	}
	as, err := s.anomalies.List(ctx, a.ID, AnomalyFilter{})
	if err != nil {
		return nil, err
	}
	return DeriveStatuses(as), nil
}

// MeterService serves the meter catalog: list, detail, readings and events.
type MeterService struct {
	meters   MeterRepository
	readings ReadingRepository
	events   EventRepository
	statuses MeterStatusSource
	cfg      engine.Config
}

// NewMeterService wires the service to its ports.
func NewMeterService(m MeterRepository, r ReadingRepository, e EventRepository, s MeterStatusSource, cfg engine.Config) *MeterService {
	return &MeterService{meters: m, readings: r, events: e, statuses: s, cfg: cfg}
}

func parseStatus(s string) (MeterStatus, error) {
	for _, st := range []MeterStatus{StatusOK, StatusAlert, StatusCritical} {
		if strings.EqualFold(s, string(st)) {
			return st, nil
		}
	}
	return "", invalid("invalid status %q: expected OK, Alert or Critical", s)
}

func statusRank(s MeterStatus) int {
	switch s {
	case StatusCritical:
		return 2
	case StatusAlert:
		return 1
	}
	return 0
}

// List returns the meters matching q, sorted as requested.
func (s *MeterService) List(ctx context.Context, q ListQuery) ([]MeterRow, error) {
	var wantStatus MeterStatus
	if q.Status != "" {
		st, err := parseStatus(q.Status)
		if err != nil {
			return nil, err
		}
		wantStatus = st
	}
	sortKey, order := q.Sort, q.Order
	if sortKey == "" {
		sortKey = SortMeterID
	}
	switch sortKey {
	case SortMeterID, SortVariation, SortKWh, SortStatus:
	default:
		return nil, invalid("invalid sort %q: expected meter_id, variation, kwh or status", q.Sort)
	}
	if order == "" {
		order = OrderAsc
	}
	if order != OrderAsc && order != OrderDesc {
		return nil, invalid("invalid order %q: expected asc or desc", q.Order)
	}

	meters, err := s.meters.List(ctx)
	if err != nil {
		return nil, err
	}
	series, err := s.readings.AllByMeter(ctx)
	if err != nil {
		return nil, err
	}
	findings, err := s.statuses.Statuses(ctx)
	if err != nil {
		return nil, err
	}

	needle := strings.ToLower(q.Q)
	var rows []MeterRow
	for _, m := range meters {
		if needle != "" && !strings.Contains(strings.ToLower(m.MeterID), needle) {
			continue
		}
		row := s.row(m.MeterID, series[m.MeterID], findings[m.MeterID])
		if wantStatus != "" && row.Status != wantStatus {
			continue
		}
		rows = append(rows, row)
	}
	sortRows(rows, sortKey, order == OrderDesc)
	return rows, nil
}

// Detail returns one meter with its findings count; ErrNotFound if unknown.
func (s *MeterService) Detail(ctx context.Context, id string) (MeterDetail, error) {
	if err := s.requireMeter(ctx, id); err != nil {
		return MeterDetail{}, err
	}
	series, err := s.readings.ByMeter(ctx, id, nil, nil)
	if err != nil {
		return MeterDetail{}, err
	}
	findings, err := s.statuses.Statuses(ctx)
	if err != nil {
		return MeterDetail{}, err
	}
	f := findings[id]
	return MeterDetail{MeterRow: s.row(id, series, f), FindingsCount: f.AnomalyCount}, nil
}

// Readings returns the readings of a meter in ascending order with their
// baseline. from and to are inclusive; the baseline always comes from the
// meter's full series.
func (s *MeterService) Readings(ctx context.Context, id string, from, to *time.Time) ([]ReadingPoint, error) {
	if err := s.requireMeter(ctx, id); err != nil {
		return nil, err
	}
	if from != nil && to != nil && from.After(*to) {
		return nil, invalid("invalid range: from is after to")
	}
	full, err := s.readings.ByMeter(ctx, id, nil, nil)
	if err != nil {
		return nil, err
	}
	base := engine.BuildBaseline(full, s.cfg)
	rs := full
	if from != nil || to != nil {
		if rs, err = s.readings.ByMeter(ctx, id, from, to); err != nil {
			return nil, err
		}
	}
	out := make([]ReadingPoint, len(rs))
	for i, r := range rs {
		out[i] = ReadingPoint{Reading: r, Baseline: base.At(r.TS)}
	}
	return out, nil
}

// Events returns the events of a meter; ErrNotFound if the meter is unknown.
func (s *MeterService) Events(ctx context.Context, id string) ([]domain.Event, error) {
	if err := s.requireMeter(ctx, id); err != nil {
		return nil, err
	}
	return s.events.ByMeter(ctx, id)
}

func (s *MeterService) requireMeter(ctx context.Context, id string) error {
	ok, err := s.meters.Exists(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("meter %s %w", id, ErrNotFound)
	}
	return nil
}

func (s *MeterService) row(id string, series []domain.Reading, f MeterFinding) MeterRow {
	row := MeterRow{MeterID: id, Status: f.Status}
	if row.Status == "" {
		row.Status = StatusOK
	}
	for _, r := range series {
		row.TotalKWh += r.KWh
	}
	if n := len(series); n > 0 {
		last := series[n-1].TS
		row.LastReadingAt = &last
	}
	row.VariationPct = latestFullDayVariation(series, engine.BuildBaseline(series, s.cfg))
	return row
}

// latestFullDayVariation compares the latest calendar day (UTC) that holds 24
// readings against the baseline daily total. It returns nil when the baseline
// is unusable or no full day exists.
func latestFullDayVariation(series []domain.Reading, b engine.Baseline) *float64 {
	if b.InsufficientHistory || b.DailyKWh <= 0 {
		return nil
	}
	type day struct {
		n   int
		kwh float64
	}
	days := map[string]*day{}
	var latest string
	for _, r := range series {
		key := r.TS.UTC().Format("2006-01-02")
		d := days[key]
		if d == nil {
			d = &day{}
			days[key] = d
		}
		d.n++
		d.kwh += r.KWh
	}
	for key, d := range days {
		if d.n == 24 && key > latest {
			latest = key
		}
	}
	if latest == "" {
		return nil
	}
	v := (days[latest].kwh - b.DailyKWh) / b.DailyKWh * 100
	return &v
}

func sortRows(rows []MeterRow, key string, desc bool) {
	less := func(a, b MeterRow) bool {
		switch key {
		case SortVariation:
			return variationOf(a) < variationOf(b)
		case SortKWh:
			return a.TotalKWh < b.TotalKWh
		case SortStatus:
			return statusRank(a.Status) < statusRank(b.Status)
		}
		return a.MeterID < b.MeterID
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if less(a, b) == less(b, a) { // tie: keep meter_id ascending
			return a.MeterID < b.MeterID
		}
		if desc {
			return less(b, a)
		}
		return less(a, b)
	})
}

// variationOf orders meters without a variation below every real value.
func variationOf(r MeterRow) float64 {
	if r.VariationPct == nil {
		return -1e18
	}
	return *r.VariationPct
}
