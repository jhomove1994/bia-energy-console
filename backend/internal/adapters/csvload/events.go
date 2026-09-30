package csvload

import (
	"errors"
	"io"
	"log/slog"
	"strings"

	"energy/internal/domain"
)

// ParseEvents parses events.csv content. Unrecognized types are stored as
// UNKNOWN with their original description.
func ParseEvents(r io.Reader, file string, log *slog.Logger) ([]domain.Event, int, error) {
	var rows []domain.Event
	skipped, err := scan(r, file, 4, log, func(rec []string) error {
		ts, err := parseTime(rec[1])
		if err != nil {
			return err
		}
		id := strings.TrimSpace(rec[0])
		if id == "" {
			return errors.New("meter_id: empty")
		}
		rows = append(rows, domain.Event{
			MeterID:     id,
			TS:          ts,
			Type:        domain.NormalizeEventType(rec[2]),
			Description: strings.TrimSpace(rec[3]),
		})
		return nil
	})
	return rows, skipped, err
}
