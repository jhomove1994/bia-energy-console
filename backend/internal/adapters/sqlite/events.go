package sqlite

import (
	"context"
	"fmt"

	"energy/internal/domain"
)

// InsertEvents inserts events keyed by (meter_id, ts, type); existing rows are
// left untouched. It returns how many rows were new.
func (w *Writer) InsertEvents(ctx context.Context, es []domain.Event) (int, error) {
	stmt, err := w.tx.PrepareContext(ctx, `INSERT OR IGNORE INTO events (meter_id, ts, type, description) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return 0, fmt.Errorf("prepare events insert: %w", err)
	}
	defer stmt.Close()
	inserted := 0
	for _, e := range es {
		res, err := stmt.ExecContext(ctx, e.MeterID, formatTime(e.TS), string(e.Type), e.Description)
		if err != nil {
			return inserted, fmt.Errorf("insert event %s %s: %w", e.MeterID, formatTime(e.TS), err)
		}
		n, _ := res.RowsAffected()
		inserted += int(n)
	}
	return inserted, nil
}
