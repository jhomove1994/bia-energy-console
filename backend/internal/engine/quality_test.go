package engine

import (
	"testing"
	"time"

	"energy/internal/domain"
)

// atAbs applies fn to every reading, giving it the absolute hour index since
// the start of the series.
func atAbs(fn func(abs int, r *domain.Reading)) func(d, h int, r *domain.Reading) {
	return func(d, h int, r *domain.Reading) { fn(d*24+h, r) }
}

func ofKind(sigs []Signal, k SignalKind) []Signal {
	var out []Signal
	for _, s := range sigs {
		if s.Kind == k {
			out = append(out, s)
		}
	}
	return out
}

func TestVoltageToggleIsDataQualityCluster(t *testing.T) {
	cfg := DefaultConfig()
	// Day 8 toggles 240 V / 202 V every 3 hours.
	series := synth(10, flat20(), func(d, h int, r *domain.Reading) {
		if d == 8 {
			r.VoltageV = 240
			if (h/3)%2 == 1 {
				r.VoltageV = 202
			}
		}
	})
	got := DetectJumps(series, cfg)
	clusters := ofKind(got, KindDQCluster)
	if len(clusters) != 1 {
		t.Fatalf("clusters = %+v", got)
	}
	c := clusters[0]
	if c.Role != RoleSeed || c.Jumps["voltage_v"] < cfg.DQMinJumps {
		t.Fatalf("cluster = %+v", c)
	}
	if len(ofKind(got, KindJump)) != 0 {
		t.Fatalf("oscillating jumps must not also be reported as single jumps: %+v", got)
	}
	// 240 <-> 202 is a swing of 38 V.
	if c.Count != c.Jumps["voltage_v"] || c.Count < 6 {
		t.Fatalf("count = %d jumps = %v", c.Count, c.Jumps)
	}
}

func TestSingleCurrentStepIsNotDataQuality(t *testing.T) {
	cfg := DefaultConfig()
	// Current steps from 100 A to 210 A (+110%) at day 8 hour 5 and stays.
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		if abs >= 8*24+5 {
			r.CurrentA = 210
		}
	}))
	got := DetectJumps(series, cfg)
	if len(ofKind(got, KindDQCluster)) != 0 {
		t.Fatalf("single step is a cluster: %+v", got)
	}
	jumps := ofKind(got, KindJump)
	if len(jumps) != 1 {
		t.Fatalf("jumps = %+v", got)
	}
	j := jumps[0]
	if j.Metric != "current_a" || j.Sign != 1 || j.Role != RoleSecondary {
		t.Fatalf("jump = %+v", j)
	}
	if !j.Start.Equal(testStart.AddDate(0, 0, 8).Add(5*hour)) || !j.End.Equal(j.Start) {
		t.Fatalf("jump time = %v..%v", j.Start, j.End)
	}
	near(t, "previous", j.Baseline, synthCurrentScale(20)*100)
	near(t, "observed", j.Observed, 210)
}

// synthCurrentScale returns synth's current for a 20 kWh reading divided by
// 100, so tests can express the healthy current without repeating the formula.
func synthCurrentScale(kwh float64) float64 {
	return kwh * 1000 / (synthRatio * synthV * synthPF) / 100
}

func TestSameDirectionVoltageJumpsAreNotDataQuality(t *testing.T) {
	cfg := DefaultConfig()
	// Voltage ramps 200 -> 210 -> 220 -> 230 in +10 V steps: three jumps, no
	// reversal.
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		base := 8*24 + 4
		switch {
		case abs < base:
			r.VoltageV = 200
		case abs < base+4:
			r.VoltageV = 210
		case abs < base+8:
			r.VoltageV = 220
		default:
			r.VoltageV = 230
		}
	}))
	got := DetectJumps(series, cfg)
	if len(ofKind(got, KindDQCluster)) != 0 {
		t.Fatalf("ramp is a cluster: %+v", got)
	}
	jumps := ofKind(got, KindJump)
	if len(jumps) != 3 {
		t.Fatalf("jumps = %d, want 3: %+v", len(jumps), jumps)
	}
}

func TestDownThenUpIsOneReversalAndNotDataQuality(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		if abs >= 8*24+3 && abs < 8*24+9 {
			r.VoltageV = 200
		}
	}))
	got := DetectJumps(series, cfg)
	if len(ofKind(got, KindDQCluster)) != 0 {
		t.Fatalf("down/up is a cluster: %+v", got)
	}
	if n := len(ofKind(got, KindJump)); n != 2 {
		t.Fatalf("jumps = %d, want 2", n)
	}
}

func TestQuantizedPowerFactorJumpsAreCounted(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		if abs >= 8*24 && abs < 8*24+16 {
			r.PF = 0.98
			if (abs/4)%2 == 1 {
				r.PF = 0.58
			}
		}
	}))
	clusters := ofKind(DetectJumps(series, cfg), KindDQCluster)
	if len(clusters) != 1 || clusters[0].Jumps["power_factor"] < cfg.DQMinJumps {
		t.Fatalf("clusters = %+v", clusters)
	}
}

func TestSmallStepsBelowThresholdsAreNotJumps(t *testing.T) {
	cfg := DefaultConfig()
	// 8 V exactly is not "> 8"; current +40% and PF 0.10 are below thresholds.
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		if abs >= 8*24+2 && abs < 8*24+4 {
			r.VoltageV = synthV + 8
			r.CurrentA *= 1.4
			r.PF = synthPF + 0.10
		}
	}))
	if got := DetectJumps(series, cfg); len(got) != 0 {
		t.Fatalf("signals = %+v", got)
	}
}

func oscillate(metric string, from, to, every int) func(abs int, r *domain.Reading) {
	return func(abs int, r *domain.Reading) {
		if abs < from || abs >= to {
			return
		}
		high := ((abs-from)/every)%2 == 1
		switch metric {
		case "voltage_v":
			r.VoltageV = 240
			if high {
				r.VoltageV = 202
			}
		case "current_a":
			r.CurrentA = 100
			if high {
				r.CurrentA = 30
			}
		}
	}
}

func TestClusterWindowsMergeAcrossMetricsWithin24Hours(t *testing.T) {
	cfg := DefaultConfig()
	base := 6 * 24
	// Voltage oscillates around base+2..base+9; the current starts 20 hours
	// after the voltage stopped.
	series := synth(12, flat20(), atAbs(func(abs int, r *domain.Reading) {
		oscillate("voltage_v", base+2, base+10, 2)(abs, r)
		oscillate("current_a", base+30, base+38, 2)(abs, r)
	}))
	clusters := ofKind(DetectJumps(series, cfg), KindDQCluster)
	if len(clusters) != 1 {
		t.Fatalf("clusters = %+v", clusters)
	}
	c := clusters[0]
	if c.Jumps["voltage_v"] == 0 || c.Jumps["current_a"] == 0 {
		t.Fatalf("jumps = %v", c.Jumps)
	}
	if want := testStart.Add(time.Duration(base+2) * hour); c.Start.Before(want) {
		t.Fatalf("start = %v", c.Start)
	}
	if c.Hours != int(c.End.Sub(c.Start)/hour)+1 {
		t.Fatalf("hours = %d for %v..%v", c.Hours, c.Start, c.End)
	}
}

func TestClusterWindowsFarApartStaySeparate(t *testing.T) {
	cfg := DefaultConfig()
	base := 4 * 24
	series := synth(14, flat20(), atAbs(func(abs int, r *domain.Reading) {
		oscillate("voltage_v", base+2, base+10, 2)(abs, r)
		oscillate("current_a", base+80, base+88, 2)(abs, r)
	}))
	clusters := ofKind(DetectJumps(series, cfg), KindDQCluster)
	if len(clusters) != 2 {
		t.Fatalf("clusters = %d, want 2", len(clusters))
	}
	if !clusters[0].Start.Before(clusters[1].Start) {
		t.Fatalf("clusters not ordered by start")
	}
}

func TestJumpsAcrossAGapAreIgnored(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), atAbs(func(abs int, r *domain.Reading) {
		if abs >= 8*24+5 {
			r.VoltageV = 200
		}
	}))
	// Drop the reading just before the step so the two neighbors are 2 h apart.
	cut := 8*24 + 4
	series = append(series[:cut:cut], series[cut+1:]...)
	if got := DetectJumps(series, cfg); len(got) != 0 {
		t.Fatalf("jump measured across a gap: %+v", got)
	}
}

func ratioSignals(series []domain.Reading, cfg Config) []Signal {
	return DetectRatio(series, BuildBaseline(series, cfg), cfg)
}

func TestStableRatioShiftIsCorroborationNotDataQuality(t *testing.T) {
	cfg := DefaultConfig()
	// kWh +25% with V, A and PF unchanged: the ratio shifts +25% for 12 hours.
	series := synth(10, flat20(), hoursAt(8, span(2, 13), scaleKWh(1.25)))
	got := ratioSignals(series, cfg)
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	s := got[0]
	if s.Kind != KindRatioStable || s.Role != RoleSecondary || s.Sign != 1 {
		t.Fatalf("signal = %+v", s)
	}
	if s.Hours != 12 || !s.Standalone {
		t.Fatalf("hours/standalone = %d/%v", s.Hours, s.Standalone)
	}
	near(t, "baseline ratio", s.Baseline, synthRatio)
	near(t, "delta pct", s.DeltaPct, 25)
}

func TestShortStableRatioRunIsNotStandalone(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), hoursAt(8, span(2, 5), scaleKWh(1.25)))
	got := ratioSignals(series, cfg)
	if len(got) != 1 || got[0].Kind != KindRatioStable {
		t.Fatalf("signals = %+v", got)
	}
	if got[0].Hours != 4 || got[0].Standalone {
		t.Fatalf("hours/standalone = %d/%v, want 4/false (below RatioStandaloneMinHours)", got[0].Hours, got[0].Standalone)
	}
	// Lowering the standalone threshold flips the flag: it reads from Config.
	cfg.RatioStandaloneMinHours = 4
	if got := ratioSignals(series, cfg); !got[0].Standalone {
		t.Fatalf("standalone ignored the configured threshold")
	}
}

func TestDeviantRatioOfBothSignsWithin24HoursIsUnstable(t *testing.T) {
	cfg := DefaultConfig()
	// Ratio x4.22 for three hours and x0.58 for three hours, same day.
	series := synth(10, flat20(), func(d, h int, r *domain.Reading) {
		hoursAt(8, span(2, 4), scaleKWh(4.22))(d, h, r)
		hoursAt(8, span(8, 10), scaleKWh(0.58))(d, h, r)
	})
	got := ratioSignals(series, cfg)
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	s := got[0]
	if s.Kind != KindRatioUnstable || s.Role != RoleSeed || s.Sign != 0 {
		t.Fatalf("signal = %+v", s)
	}
	if s.Flagged != 6 {
		t.Fatalf("flagged = %d, want 6", s.Flagged)
	}
	if !s.Start.Equal(testStart.AddDate(0, 0, 8).Add(2*hour)) || !s.End.Equal(testStart.AddDate(0, 0, 8).Add(10*hour)) {
		t.Fatalf("window = %v..%v", s.Start, s.End)
	}
}

func TestSameSignDeviationsAreNeverUnstable(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), hoursAt(8, []int{2, 5, 9, 14}, scaleKWh(3)))
	for _, s := range ratioSignals(series, cfg) {
		if s.Kind == KindRatioUnstable {
			t.Fatalf("same-sign deviations are unstable: %+v", s)
		}
	}
}

func TestRatioSkipsRowsWithNonPositiveDenominator(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), hoursAt(8, span(2, 8), func(r *domain.Reading) {
		r.PF = 0
		r.KWh = 80
	}))
	if got := ratioSignals(series, cfg); len(got) != 0 {
		t.Fatalf("signals = %+v", got)
	}
}

func TestRatioSkippedWithoutBaselineRatio(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, flat20(), func(d, h int, r *domain.Reading) {
		if d < 7 {
			r.PF = 0 // no ratio can be computed in the baseline window
		}
		if d == 8 {
			r.KWh *= 3
		}
	})
	if got := ratioSignals(series, cfg); len(got) != 0 {
		t.Fatalf("signals = %+v", got)
	}
}
