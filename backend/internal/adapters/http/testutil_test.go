package httpapi

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"energy/internal/adapters/sqlite"
	"energy/internal/app"
	"energy/internal/domain"
	"energy/internal/engine"
)

var fixtureStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// fixture is a real SQLite database seeded with synthetic data plus a router
// wired to it.
type fixture struct {
	Handler http.Handler
	Token   string
	Logs    *bytes.Buffer
}

// fixtureKWh is the consumption of meter idx (0-based) on a day and hour: a
// gentle daily shape, with meter M-109 (idx 8) running at 2.09x on the last
// day.
func fixtureKWh(idx, day, hour int) float64 {
	v := 10 + float64(hour%4)
	if idx == 8 && day == 13 {
		v *= 2.09
	}
	return v
}

// newFixture seeds 12 meters M-101..M-112 with 14 days of hourly readings and
// one event on M-109.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	ctx := context.Background()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "api.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := sqlite.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}

	var ids []string
	var rs []domain.Reading
	for i := 0; i < 12; i++ {
		id := fmt.Sprintf("M-%d", 101+i)
		ids = append(ids, id)
		for d := 0; d < 14; d++ {
			for h := 0; h < 24; h++ {
				rs = append(rs, domain.Reading{
					MeterID: id, TS: fixtureStart.AddDate(0, 0, d).Add(time.Duration(h) * time.Hour),
					KWh: fixtureKWh(i, d, h), VoltageV: 230, CurrentA: 100, PF: 0.9, Status: domain.StatusOK,
				})
			}
		}
	}
	events := []domain.Event{{MeterID: "M-109", TS: fixtureStart.AddDate(0, 0, 12).Add(14 * time.Hour), Type: domain.EventUnknown, Description: "firmware update"}}
	err = db.WriteTx(ctx, func(w *sqlite.Writer) error {
		if _, err := w.InsertMeters(ctx, ids, fixtureStart); err != nil {
			return err
		}
		if _, err := w.InsertReadings(ctx, rs); err != nil {
			return err
		}
		_, err := w.InsertEvents(ctx, events)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}

	meters, readings, evs := sqlite.NewMeterStore(db), sqlite.NewReadingStore(db), sqlite.NewEventStore(db)
	now := fixtureStart.AddDate(0, 0, 20)
	tokens := NewTokens(testSecret, time.Hour, fixedClock(now))
	var logs bytes.Buffer
	h := NewRouter(Deps{
		DemoUser: testUser, DemoPassword: testPass, Tokens: tokens,
		Log:       slog.New(slog.NewTextHandler(&logs, nil)),
		Meters:    app.NewMeterService(meters, readings, evs, app.NoAnalysis{}, engine.DefaultConfig()),
		Dashboard: app.NewDashboardService(meters, readings, app.NoLatestDone{}),
	})
	token, _ := tokens.Issue(testUser)
	return &fixture{Handler: h, Token: token, Logs: &logs}
}
