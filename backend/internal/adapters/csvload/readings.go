// Package csvload parses the source CSV files and seeds them into the store.
package csvload

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"energy/internal/domain"
)

var timeLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04", time.RFC3339, "2006-01-02T15:04:05"}

// parseTime accepts the supported layouts; timezone-less values are UTC.
func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range timeLayouts {
		if t, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparsable timestamp %q", s)
}

// scan reads records after the header. handle returns a reason when the row
// must be skipped; skipped rows are logged with file, line and reason.
func scan(r io.Reader, file string, wantCols int, log *slog.Logger, handle func(rec []string) error) (int, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	skipped := 0
	first := true
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return skipped, nil
		}
		line, _ := cr.FieldPos(0)
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				skipped++
				log.Warn("csv row skipped", "file", file, "line", pe.StartLine, "reason", pe.Err.Error())
				continue
			}
			return skipped, fmt.Errorf("read %s: %w", file, err)
		}
		if first {
			first = false // header row
			continue
		}
		if len(rec) != wantCols {
			skipped++
			log.Warn("csv row skipped", "file", file, "line", line, "reason", fmt.Sprintf("expected %d columns, got %d", wantCols, len(rec)))
			continue
		}
		if err := handle(rec); err != nil {
			skipped++
			log.Warn("csv row skipped", "file", file, "line", line, "reason", err.Error())
		}
	}
}

func parseFloat(field, value string) (float64, error) {
	f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0, fmt.Errorf("%s: non-numeric value %q", field, value)
	}
	return f, nil
}

// ParseReadings parses readings.csv content. Malformed rows are skipped,
// logged, and counted; valid rows are returned unchanged (no interpolation).
func ParseReadings(r io.Reader, file string, log *slog.Logger) ([]domain.Reading, int, error) {
	var rows []domain.Reading
	skipped, err := scan(r, file, 7, log, func(rec []string) error {
		ts, err := parseTime(rec[1])
		if err != nil {
			return fmt.Errorf("timestamp: %w", err)
		}
		var nums [4]float64
		for i, name := range []string{"consumption_kwh", "voltage_v", "current_a", "power_factor"} {
			if nums[i], err = parseFloat(name, rec[2+i]); err != nil {
				return err
			}
		}
		id := strings.TrimSpace(rec[0])
		if id == "" {
			return errors.New("meter_id: empty")
		}
		rows = append(rows, domain.Reading{
			MeterID: id, TS: ts,
			KWh: nums[0], VoltageV: nums[1], CurrentA: nums[2], PF: nums[3],
			Status: strings.TrimSpace(rec[6]),
		})
		return nil
	})
	return rows, skipped, err
}
