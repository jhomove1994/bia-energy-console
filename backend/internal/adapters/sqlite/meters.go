package sqlite

import (
	"context"
	"fmt"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

// InsertMeters inserts meters that do not exist yet and returns how many were new.
func (w *Writer) InsertMeters(ctx context.Context, ids []string, createdAt time.Time) (int, error) {
	stmt, err := w.tx.PrepareContext(ctx, "INSERT OR IGNORE INTO meters (meter_id, created_at) VALUES (?, ?)")
	if err != nil {
		return 0, fmt.Errorf("prepare meters insert: %w", err)
	}
	defer stmt.Close()
	inserted := 0
	for _, id := range ids {
		res, err := stmt.ExecContext(ctx, id, formatTime(createdAt))
		if err != nil {
			return inserted, fmt.Errorf("insert meter %s: %w", id, err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}
	return inserted, nil
}

// MeterStore reads the meter catalog.
type MeterStore struct{ db *DB }

var _ app.MeterRepository = (*MeterStore)(nil)

// NewMeterStore returns a MeterStore over db.
func NewMeterStore(db *DB) *MeterStore { return &MeterStore{db: db} }

// List returns every meter ordered by meter id.
func (s *MeterStore) List(ctx context.Context) ([]domain.Meter, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT meter_id, created_at FROM meters ORDER BY meter_id")
	if err != nil {
		return nil, fmt.Errorf("list meters: %w", err)
	}
	defer rows.Close()
	var out []domain.Meter
	for rows.Next() {
		var m domain.Meter
		var created string
		if err := rows.Scan(&m.MeterID, &created); err != nil {
			return nil, fmt.Errorf("scan meter: %w", err)
		}
		if m.CreatedAt, err = parseTime(created); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// Exists reports whether a meter with the given id is stored.
func (s *MeterStore) Exists(ctx context.Context, meterID string) (bool, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, "SELECT count(*) FROM meters WHERE meter_id = ?", meterID).Scan(&n); err != nil {
		return false, fmt.Errorf("meter exists %s: %w", meterID, err)
	}
	return n > 0, nil
}
