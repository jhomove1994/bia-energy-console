package csvload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"energy/internal/adapters/sqlite"
	"energy/internal/domain"
)

// ErrDataMissing is wrapped when a required CSV file is absent.
var ErrDataMissing = errors.New("data file missing")

// Result summarizes one seed run.
type Result struct {
	Meters, Readings, Events int
	Inserted                 int
	Skipped                  int
	Gaps                     map[string]int // missing hourly timestamps per meter
	NonOK                    int            // readings whose status is not OK
}

const (
	readingsFile = "readings.csv"
	eventsFile   = "events.csv"
)

// Seed loads readings.csv and events.csv from dataDir into db in one
// transaction. It is idempotent: existing rows are never overwritten.
func Seed(ctx context.Context, db *sqlite.DB, dataDir string, log *slog.Logger) (Result, error) {
	readings, skippedR, err := loadFile(dataDir, readingsFile, log, ParseReadings)
	if err != nil {
		return Result{}, err
	}
	events, skippedE, err := loadFile(dataDir, eventsFile, log, ParseEvents)
	if err != nil {
		return Result{}, err
	}

	meterSet := map[string]bool{}
	for _, r := range readings {
		meterSet[r.MeterID] = true
	}
	ids := make([]string, 0, len(meterSet))
	for id := range meterSet {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	kept := events[:0:0]
	for _, e := range events {
		if !meterSet[e.MeterID] {
			skippedE++
			log.Warn("event skipped", "file", eventsFile, "meter", e.MeterID, "reason", "meter has no readings")
			continue
		}
		kept = append(kept, e)
	}

	res := Result{
		Meters: len(ids), Readings: len(readings), Events: len(kept),
		Skipped: skippedR + skippedE,
		Gaps:    cadenceGaps(readings),
	}
	for _, r := range readings {
		if r.Status != domain.StatusOK {
			res.NonOK++
		}
	}

	err = db.WriteTx(ctx, func(w *sqlite.Writer) error {
		m, err := w.InsertMeters(ctx, ids, time.Now().UTC())
		if err != nil {
			return err
		}
		r, err := w.InsertReadings(ctx, readings)
		if err != nil {
			return err
		}
		e, err := w.InsertEvents(ctx, kept)
		if err != nil {
			return err
		}
		res.Inserted = m + r + e
		return nil
	})
	if err != nil {
		return Result{}, fmt.Errorf("seed transaction: %w", err)
	}

	totalGaps := 0
	for _, id := range ids {
		if g := res.Gaps[id]; g > 0 {
			totalGaps += g
			log.Warn("seed: cadence gaps", "meter", id, "gaps", g)
		}
	}
	log.Info(fmt.Sprintf("seed: meters=%d readings=%d events=%d (inserted=%d)", res.Meters, res.Readings, res.Events, res.Inserted),
		"skipped_rows", res.Skipped, "gaps", totalGaps, "non_ok", res.NonOK)
	return res, nil
}

func loadFile[T any](dir, name string, log *slog.Logger, parse func(r io.Reader, file string, log *slog.Logger) ([]T, int, error)) ([]T, int, error) {
	path := filepath.Join(dir, name)
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, 0, fmt.Errorf("%w: %s", ErrDataMissing, path)
		}
		return nil, 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	return parse(f, name, log)
}

// cadenceGaps counts missing hourly timestamps between consecutive readings
// of each meter. Nothing is interpolated or stored.
func cadenceGaps(rs []domain.Reading) map[string]int {
	byMeter := map[string][]time.Time{}
	for _, r := range rs {
		byMeter[r.MeterID] = append(byMeter[r.MeterID], r.TS)
	}
	gaps := map[string]int{}
	for id, ts := range byMeter {
		sort.Slice(ts, func(i, j int) bool { return ts[i].Before(ts[j]) })
		n := 0
		for i := 1; i < len(ts); i++ {
			if d := ts[i].Sub(ts[i-1]); d > time.Hour {
				n += int(d/time.Hour) - 1
			}
		}
		gaps[id] = n
	}
	return gaps
}
