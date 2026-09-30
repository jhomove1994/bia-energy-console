package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"energy/internal/app"
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

// EventStore reads operational events.
type EventStore struct{ db *DB }

var _ app.EventRepository = (*EventStore)(nil)

// NewEventStore returns an EventStore over db.
func NewEventStore(db *DB) *EventStore { return &EventStore{db: db} }

func scanEvents(rows *sql.Rows) ([]domain.Event, error) {
	defer rows.Close()
	var out []domain.Event
	for rows.Next() {
		var e domain.Event
		var ts, typ string
		if err := rows.Scan(&e.ID, &e.MeterID, &ts, &typ, &e.Description); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		var err error
		if e.TS, err = parseTime(ts); err != nil {
			return nil, err
		}
		e.Type = domain.EventType(typ)
		out = append(out, e)
	}
	return out, rows.Err()
}

// All returns every event ordered by meter and time.
func (s *EventStore) All(ctx context.Context) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, meter_id, ts, type, description FROM events ORDER BY meter_id, ts, id")
	if err != nil {
		return nil, fmt.Errorf("all events: %w", err)
	}
	return scanEvents(rows)
}

// ByMeter returns one meter's events ascending by time.
func (s *EventStore) ByMeter(ctx context.Context, meterID string) ([]domain.Event, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT id, meter_id, ts, type, description FROM events WHERE meter_id = ? ORDER BY ts, id", meterID)
	if err != nil {
		return nil, fmt.Errorf("events by meter %s: %w", meterID, err)
	}
	return scanEvents(rows)
}
