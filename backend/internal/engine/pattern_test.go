package engine

import (
	"testing"

	"energy/internal/domain"
)

func patternBreaks(series []domain.Reading, cfg Config) []Signal {
	return DetectPatternBreaks(series, BuildBaseline(series, cfg), cfg)
}

// replaceDay overwrites the kWh of a whole day with f(hour).
func replaceDay(day int, f func(h int) float64) func(d, h int, r *domain.Reading) {
	return func(d, h int, r *domain.Reading) {
		if d == day {
			r.KWh = f(h)
		}
	}
}

func TestUniformStepIsNotAPatternBreak(t *testing.T) {
	prof := shapedProfile()
	// A uniform +47% step with a little hourly noise: correlation stays near
	// 1 and the normalized shape barely moves.
	series := synth(10, prof, replaceDay(8, func(h int) float64 {
		noise := 1 + 0.03*float64((h%3)-1)
		return prof[h] * 1.47 * noise
	}))
	if got := patternBreaks(series, DefaultConfig()); len(got) != 0 {
		t.Fatalf("uniform step flagged: %+v", got)
	}
}

func TestReshapedDayIsAPatternBreak(t *testing.T) {
	prof := shapedProfile()
	series := synth(10, prof, replaceDay(8, func(h int) float64 { return prof[23-h] }))
	got := patternBreaks(series, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	s := got[0]
	if s.Kind != KindPatternBreak || s.Role != RoleSecondary || s.Hours != 24 {
		t.Fatalf("signal = %+v", s)
	}
	if s.Corr >= 0.80 || s.Shape <= 0.15 {
		t.Fatalf("corr/shape = %v/%v", s.Corr, s.Shape)
	}
	if !s.Start.Equal(testStart.AddDate(0, 0, 8)) || s.MeterID != "X-1" {
		t.Fatalf("signal = %+v", s)
	}
}

func TestPatternBreakNeedsBothCorrelationAndShape(t *testing.T) {
	prof := shapedProfile()
	var mean float64
	for _, v := range prof {
		mean += v / 24
	}

	// Amplified peaks keep the correlation at 1: shape alone is not a break.
	amplified := synth(10, prof, replaceDay(8, func(h int) float64 { return mean + 1.6*(prof[h]-mean) }))
	if got := patternBreaks(amplified, DefaultConfig()); len(got) != 0 {
		t.Fatalf("shape-only deviation flagged: %+v", got)
	}

	// The reversed day is a break with defaults, but not when the shape
	// threshold is raised: both conditions read from Config.
	reversed := synth(10, prof, replaceDay(8, func(h int) float64 { return prof[23-h] }))
	cfg := DefaultConfig()
	cfg.PatternShapeMin = 10
	if got := patternBreaks(reversed, cfg); len(got) != 0 {
		t.Fatalf("shape threshold ignored: %+v", got)
	}
	cfg = DefaultConfig()
	cfg.PatternCorrMax = -2
	if got := patternBreaks(reversed, cfg); len(got) != 0 {
		t.Fatalf("correlation threshold ignored: %+v", got)
	}
}

func TestPartialDaysAreSkippedByPatternDetector(t *testing.T) {
	prof := shapedProfile()
	series := synth(10, prof, replaceDay(8, func(h int) float64 { return prof[23-h] }))
	// Remove one reading of the reversed day.
	cut := 8*24 + 11
	series = append(series[:cut:cut], series[cut+1:]...)
	if got := patternBreaks(series, DefaultConfig()); len(got) != 0 {
		t.Fatalf("partial day flagged: %+v", got)
	}
}

func TestFlatProfileHasNoDefinedCorrelation(t *testing.T) {
	series := synth(10, flatProfile(20), replaceDay(8, func(h int) float64 { return float64(10 + h) }))
	if got := patternBreaks(series, DefaultConfig()); len(got) != 0 {
		t.Fatalf("flat baseline profile produced breaks: %+v", got)
	}
}
