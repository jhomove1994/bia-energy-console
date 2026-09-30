package engine

import (
	"sort"
	"time"

	"energy/internal/domain"
)

// HourStats is the robust location and dispersion of one hour of day.
type HourStats struct {
	Median float64
	MAD    float64
}

// Baseline is the expected behavior of one meter per hour of day (UTC),
// computed from the first BaselineDays of its series.
type Baseline struct {
	// InsufficientHistory is set when some hour of day has fewer than
	// BaselineMinSamples samples. The meter must be skipped, not flagged.
	InsufficientHistory bool

	KWh     [24]HourStats
	Voltage [24]HourStats
	Current [24]HourStats
	PF      [24]HourStats
	Ratio   [24]HourStats

	// SigmaKWh is the MAD-based scale used for robust z-scores, floored.
	SigmaKWh [24]float64
	// Profile is the median kWh per hour of day.
	Profile [24]float64
	// DailyKWh is the sum of the hourly median kWh.
	DailyKWh float64
	// RatioMedian is the median electrical ratio over the baseline window.
	RatioMedian float64
}

// Expected is the baseline value of every metric at one point in time.
type Expected struct {
	KWh, VoltageV, CurrentA, PF float64
}

// At returns the hour-of-day baseline that applies to t.
func (b Baseline) At(t time.Time) Expected {
	h := t.UTC().Hour()
	return Expected{
		KWh:      b.KWh[h].Median,
		VoltageV: b.Voltage[h].Median,
		CurrentA: b.Current[h].Median,
		PF:       b.PF[h].Median,
	}
}

// ElectricalRatio is kWh divided by apparent energy V*A*PF/1000. ok is false
// when the denominator is not positive.
func ElectricalRatio(r domain.Reading) (ratio float64, ok bool) {
	den := r.VoltageV * r.CurrentA * r.PF / 1000
	if den <= 0 {
		return 0, false
	}
	return r.KWh / den, true
}

// BuildBaseline computes the baseline of a single meter from series, which
// must be sorted by timestamp ascending. The window starts at the first
// reading, never at a calendar date.
func BuildBaseline(series []domain.Reading, cfg Config) Baseline {
	var b Baseline
	if len(series) == 0 {
		b.InsufficientHistory = true
		return b
	}
	end := series[0].TS.Add(time.Duration(cfg.BaselineDays) * 24 * time.Hour)

	var kwh, volt, amp, pf, ratio [24][]float64
	var allRatios []float64
	for _, r := range series {
		if !r.TS.Before(end) {
			break
		}
		h := r.TS.UTC().Hour()
		kwh[h] = append(kwh[h], r.KWh)
		volt[h] = append(volt[h], r.VoltageV)
		amp[h] = append(amp[h], r.CurrentA)
		pf[h] = append(pf[h], r.PF)
		if q, ok := ElectricalRatio(r); ok {
			ratio[h] = append(ratio[h], q)
			allRatios = append(allRatios, q)
		}
	}

	for h := 0; h < 24; h++ {
		if len(kwh[h]) < cfg.BaselineMinSamples {
			b.InsufficientHistory = true
		}
		b.KWh[h] = hourStats(kwh[h])
		b.Voltage[h] = hourStats(volt[h])
		b.Current[h] = hourStats(amp[h])
		b.PF[h] = hourStats(pf[h])
		b.Ratio[h] = hourStats(ratio[h])

		floor := cfg.MADFloorPct * b.KWh[h].Median
		b.SigmaKWh[h] = cfg.MADScale * maxFloat(b.KWh[h].MAD, floor)
		b.Profile[h] = b.KWh[h].Median
		b.DailyKWh += b.Profile[h]
	}
	b.RatioMedian = median(allRatios)
	return b
}

func hourStats(xs []float64) HourStats {
	m := median(xs)
	dev := make([]float64, len(xs))
	for i, x := range xs {
		d := x - m
		if d < 0 {
			d = -d
		}
		dev[i] = d
	}
	return HourStats{Median: m, MAD: median(dev)}
}

// median returns the median of xs without modifying it; 0 for an empty slice.
func median(xs []float64) float64 {
	n := len(xs)
	if n == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
