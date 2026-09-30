package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

// InsertReadings inserts readings keyed by (meter_id, ts); existing rows are
// left untouched. It returns how many rows were new.
func (w *Writer) InsertReadings(ctx context.Context, rs []domain.Reading) (int, error) {
	stmt, err := w.tx.PrepareContext(ctx, `INSERT OR IGNORE INTO readings
		(meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor, status)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare readings insert: %w", err)
	}
	defer stmt.Close()
	inserted := 0
	for _, r := range rs {
		res, err := stmt.ExecContext(ctx, r.MeterID, formatTime(r.TS), r.KWh, r.VoltageV, r.CurrentA, r.PF, r.Status)
		if err != nil {
			return inserted, fmt.Errorf("insert reading %s %s: %w", r.MeterID, formatTime(r.TS), err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}
	return inserted, nil
}

// ReadingStore reads hourly readings.
type ReadingStore struct{ db *DB }

var _ app.ReadingRepository = (*ReadingStore)(nil)

// NewReadingStore returns a ReadingStore over db.
func NewReadingStore(db *DB) *ReadingStore { return &ReadingStore{db: db} }

const readingColumns = "meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor, status"

func scanReadings(rows *sql.Rows) ([]domain.Reading, error) {
	defer rows.Close()
	var out []domain.Reading
	for rows.Next() {
		var r domain.Reading
		var ts string
		if err := rows.Scan(&r.MeterID, &ts, &r.KWh, &r.VoltageV, &r.CurrentA, &r.PF, &r.Status); err != nil {
			return nil, fmt.Errorf("scan reading: %w", err)
		}
		var err error
		if r.TS, err = parseTime(ts); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ByMeter returns one meter's readings ascending; from and to are inclusive.
func (s *ReadingStore) ByMeter(ctx context.Context, meterID string, from, to *time.Time) ([]domain.Reading, error) {
	q := "SELECT " + readingColumns + " FROM readings WHERE meter_id = ?"
	args := []any{meterID}
	if from != nil {
		q += " AND ts >= ?"
		args = append(args, formatTime(*from))
	}
	if to != nil {
		q += " AND ts <= ?"
		args = append(args, formatTime(*to))
	}
	rows, err := s.db.QueryContext(ctx, q+" ORDER BY ts", args...)
	if err != nil {
		return nil, fmt.Errorf("readings by meter %s: %w", meterID, err)
	}
	return scanReadings(rows)
}

// AllByMeter returns every series keyed by meter id, each sorted by timestamp.
func (s *ReadingStore) AllByMeter(ctx context.Context) (map[string][]domain.Reading, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT "+readingColumns+" FROM readings ORDER BY meter_id, ts")
	if err != nil {
		return nil, fmt.Errorf("all readings: %w", err)
	}
	all, err := scanReadings(rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]domain.Reading)
	for _, r := range all {
		out[r.MeterID] = append(out[r.MeterID], r)
	}
	return out, nil
}

// TotalKWh is the sum of consumption over all readings.
func (s *ReadingStore) TotalKWh(ctx context.Context) (float64, error) {
	var total float64
	if err := s.db.QueryRowContext(ctx, "SELECT COALESCE(SUM(consumption_kwh), 0) FROM readings").Scan(&total); err != nil {
		return 0, fmt.Errorf("total kwh: %w", err)
	}
	return total, nil
}
