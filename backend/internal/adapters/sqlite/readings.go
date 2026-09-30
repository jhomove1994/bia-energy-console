package sqlite

import (
	"context"
	"fmt"

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
