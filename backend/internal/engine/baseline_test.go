package engine

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"energy/internal/domain"
)

var testStart = time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)

// hourlySeries builds days*24 hourly readings from testStart. fn receives the
// day index and hour of day and returns the reading values (TS and meter are
// filled in here).
func hourlySeries(days int, fn func(day, hour int) domain.Reading) []domain.Reading {
	out := make([]domain.Reading, 0, days*24)
	for d := 0; d < days; d++ {
		for h := 0; h < 24; h++ {
			r := fn(d, h)
			r.MeterID = "X-1"
			r.TS = testStart.AddDate(0, 0, d).Add(time.Duration(h) * time.Hour)
			r.Status = domain.StatusOK
			out = append(out, r)
		}
	}
	return out
}

// steadyReading is a healthy reading whose electrical ratio is 1.05.
func steadyReading(kwh float64) domain.Reading {
	const v, a, pf = 230.0, 100.0, 0.9
	return domain.Reading{KWh: kwh, VoltageV: v, CurrentA: a, PF: pf}
}

func near(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestBuildBaselineMedianAndMAD(t *testing.T) {
	// Hour 0 varies 10..16 over the first week: median 13, deviations
	// 3,2,1,0,1,2,3 so MAD 2. Hour 5 has 4..10 (median 7, MAD 2). Everything
	// else is constant. Days after the first week are heavily contaminated and
	// must not influence the baseline.
	series := hourlySeries(14, func(d, h int) domain.Reading {
		switch {
		case d >= 7:
			return steadyReading(999)
		case h == 0:
			return steadyReading(float64(10 + d))
		case h == 5:
			return steadyReading(float64(4 + d))
		}
		return steadyReading(20)
	})
	b := BuildBaseline(series, DefaultConfig())
	if b.InsufficientHistory {
		t.Fatal("unexpected insufficient history")
	}
	near(t, "hour0 median", b.KWh[0].Median, 13)
	near(t, "hour0 MAD", b.KWh[0].MAD, 2)
	near(t, "hour5 median", b.KWh[5].Median, 7)
	near(t, "hour5 MAD", b.KWh[5].MAD, 2)
	near(t, "hour9 median", b.KWh[9].Median, 20)
	near(t, "hour9 MAD", b.KWh[9].MAD, 0)
	near(t, "voltage median", b.Voltage[3].Median, 230)
	near(t, "current median", b.Current[3].Median, 100)
	near(t, "pf median", b.PF[3].Median, 0.9)
}

func TestBuildBaselineSigmaHasMADFloor(t *testing.T) {
	series := hourlySeries(7, func(d, h int) domain.Reading {
		if h == 0 {
			return steadyReading(float64(10 + d))
		}
		return steadyReading(20)
	})
	b := BuildBaseline(series, DefaultConfig())
	// Hour 0: MAD 2 exceeds 3% of the median (0.39), so it is used as is.
	near(t, "sigma hour0", b.SigmaKWh[0], 1.4826*2)
	// Hour 9: MAD 0, floored to 3% of the median 20.
	near(t, "sigma hour9", b.SigmaKWh[9], 1.4826*0.03*20)
}

func TestBuildBaselineDailyProfileAndRatio(t *testing.T) {
	series := hourlySeries(7, func(d, h int) domain.Reading {
		// ratio = kwh / (230*100*0.9/1000) = (h+1) * 1.05
		return steadyReading(float64(h+1) * 20.7 * 1.05)
	})
	b := BuildBaseline(series, DefaultConfig())
	var want float64
	for h := 0; h < 24; h++ {
		near(t, "profile", b.Profile[h], float64(h+1)*20.7*1.05)
		want += float64(h+1) * 20.7 * 1.05
	}
	near(t, "daily kwh", b.DailyKWh, want)
	near(t, "ratio median", b.RatioMedian, 12.5*1.05) // median of 1.05..25.2
	near(t, "hour ratio", b.Ratio[7].Median, 8*1.05)
}

func TestBuildBaselineRatioSkipsNonPositiveDenominator(t *testing.T) {
	series := hourlySeries(7, func(d, h int) domain.Reading {
		r := steadyReading(21.735)
		if d < 3 { // 3 of 7 samples per hour have a zero current
			r.CurrentA = 0
			r.KWh = 500
		}
		return r
	})
	b := BuildBaseline(series, DefaultConfig())
	near(t, "ratio median", b.RatioMedian, 1.05)
	near(t, "hour ratio", b.Ratio[0].Median, 1.05)
}

func TestBuildBaselineInsufficientHistory(t *testing.T) {
	cfg := DefaultConfig()
	tests := []struct {
		name   string
		series []domain.Reading
		want   bool
	}{
		{"four days is too short", hourlySeries(4, func(d, h int) domain.Reading { return steadyReading(20) }), true},
		{"five days meets the minimum", hourlySeries(5, func(d, h int) domain.Reading { return steadyReading(20) }), false},
		{"one sparse hour", func() []domain.Reading {
			all := hourlySeries(7, func(d, h int) domain.Reading { return steadyReading(20) })
			var out []domain.Reading
			for _, r := range all {
				if r.TS.Hour() == 3 && r.TS.Day() <= 3 { // hour 3 keeps 4 samples
					continue
				}
				out = append(out, r)
			}
			return out
		}(), true},
		{"empty series", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildBaseline(tt.series, cfg).InsufficientHistory; got != tt.want {
				t.Fatalf("InsufficientHistory = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildBaselineIsRelativeToSeriesStart(t *testing.T) {
	fn := func(d, h int) domain.Reading { return steadyReading(float64(10 + d + h%3)) }
	base := hourlySeries(10, fn)
	shifted := make([]domain.Reading, len(base))
	for i, r := range base {
		r.TS = r.TS.AddDate(0, 0, 30)
		shifted[i] = r
	}
	if a, b := BuildBaseline(base, DefaultConfig()), BuildBaseline(shifted, DefaultConfig()); a != b {
		t.Fatalf("baseline changed after shifting the series by 30 days:\n%+v\n%+v", a, b)
	}
}

func TestBaselineAtReturnsHourOfDayValues(t *testing.T) {
	series := hourlySeries(7, func(d, h int) domain.Reading {
		r := steadyReading(float64(h) + 1)
		r.VoltageV = 200 + float64(h)
		return r
	})
	b := BuildBaseline(series, DefaultConfig())
	at := b.At(time.Date(2026, 10, 3, 6, 0, 0, 0, time.UTC))
	near(t, "kwh", at.KWh, 7)
	near(t, "voltage", at.VoltageV, 206)
	near(t, "current", at.CurrentA, 100)
	near(t, "pf", at.PF, 0.9)
}

func TestBaselineSourceDoesNotUseWallClock(t *testing.T) {
	src, err := os.ReadFile("baseline.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "time.Now") {
		t.Fatal("baseline.go must not call time.Now")
	}
}
