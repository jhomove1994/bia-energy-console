package sqlite

import (
	"context"
	"fmt"
	"time"
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
