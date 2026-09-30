package engine

import (
	"math"

	"energy/internal/domain"
)

// DetectPatternBreaks compares the hourly shape of every complete day (24
// readings) with the baseline profile. A day is a break only when both
// hold: its Pearson correlation with the profile is below PatternCorrMax and
// its normalized shape deviation is above PatternShapeMin. Normalizing by the
// day's own mean makes the test scale-invariant, so a uniform step in
// consumption is not a break. Partial days, and baselines whose profile has no
// variance, are skipped. Breaks are secondary evidence.
func DetectPatternBreaks(series []domain.Reading, b Baseline, cfg Config) []Signal {
	if b.InsufficientHistory {
		return nil
	}
	profMean := mean(b.Profile[:])
	if profMean <= 0 {
		return nil
	}
	var out []Signal
	for _, day := range dayGroups(series) {
		if len(day) != 24 {
			continue
		}
		var kwh [24]float64
		for _, r := range day {
			kwh[r.TS.UTC().Hour()] = r.KWh
		}
		corr, ok := pearson(kwh[:], b.Profile[:])
		if !ok {
			continue
		}
		dayMean := mean(kwh[:])
		if dayMean <= 0 {
			continue
		}
		var sum float64
		n := 0
		for h := 0; h < 24; h++ {
			expected := b.Profile[h] / profMean
			if expected <= 0 {
				continue
			}
			sum += math.Abs(kwh[h]/dayMean-expected) / expected
			n++
		}
		if n == 0 {
			continue
		}
		shape := sum / float64(n)
		if corr >= cfg.PatternCorrMax || shape <= cfg.PatternShapeMin {
			continue
		}
		out = append(out, Signal{
			MeterID:  day[0].MeterID,
			Kind:     KindPatternBreak,
			Metric:   "hourly_pattern",
			Start:    day[0].TS,
			End:      day[len(day)-1].TS,
			Role:     RoleSecondary,
			Hours:    len(day),
			Observed: corr,
			Unit:     "correlation",
			Corr:     corr,
			Shape:    shape,
		})
	}
	return out
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s float64
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

// pearson returns the correlation of x and y; ok is false when either has no
// variance.
func pearson(x, y []float64) (r float64, ok bool) {
	mx, my := mean(x), mean(y)
	var sxy, sxx, syy float64
	for i := range x {
		dx, dy := x[i]-mx, y[i]-my
		sxy += dx * dy
		sxx += dx * dx
		syy += dy * dy
	}
	if sxx == 0 || syy == 0 {
		return 0, false
	}
	return sxy / math.Sqrt(sxx*syy), true
}
