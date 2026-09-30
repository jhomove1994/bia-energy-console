package engine

import (
	"testing"

	"energy/internal/domain"
)

// hoursAt applies fn to the readings of day at the given hours of day.
func hoursAt(day int, hours []int, fn func(r *domain.Reading)) func(d, h int, r *domain.Reading) {
	return func(d, h int, r *domain.Reading) {
		if d != day {
			return
		}
		for _, x := range hours {
			if x == h {
				fn(r)
			}
		}
	}
}

func span(from, to int) []int {
	var out []int
	for h := from; h <= to; h++ {
		out = append(out, h)
	}
	return out
}

func scaleKWh(f float64) func(r *domain.Reading) {
	return func(r *domain.Reading) { r.KWh *= f }
}

func consumption(series []domain.Reading, cfg Config) []Signal {
	return DetectConsumption(series, BuildBaseline(series, cfg), cfg)
}

func kinds(sigs []Signal) []SignalKind {
	out := make([]SignalKind, len(sigs))
	for i, s := range sigs {
		out[i] = s.Kind
	}
	return out
}

func TestHourlyFlagRequiresBothEffectAndZ(t *testing.T) {
	cfg := DefaultConfig()
	flat := flatProfile(20)

	// +25% on a MAD-floored hour has z about 5.6 (above 3.5) but the effect is
	// below the 30% gate: nothing is flagged.
	series := synth(10, flat, hoursAt(8, span(2, 9), scaleKWh(1.25)))
	if got := consumption(series, cfg); len(got) != 0 {
		t.Fatalf("+25%% flagged: %+v", got)
	}

	// +40% on a hour whose baseline is spread out has z about 1.8: not flagged.
	deltas := []float64{-3, -3, -3, 0, 3, 3, 3}
	series = synth(10, flat, func(d, h int, r *domain.Reading) {
		if h == 10 && d < 7 {
			r.KWh = 20 + deltas[d]
		}
		if h == 10 && d == 8 {
			r.KWh = 28
		}
	})
	if got := consumption(series, cfg); len(got) != 0 {
		t.Fatalf("+40%% with low z flagged: %+v", got)
	}

	// +47% on a steady hour has a large z: flagged (single point below the
	// spike threshold and alone, so it does not become an episode).
	series = synth(10, flat, hoursAt(8, span(2, 3), scaleKWh(1.47)))
	if got := consumption(series, cfg); len(got) != 1 || got[0].Kind != KindSpike {
		t.Fatalf("+47%% for 2 h: %+v", got)
	}
}

func TestTinyMADDoesNotFire(t *testing.T) {
	cfg := DefaultConfig()
	// Every baseline sample is identical (MAD 0). A 12% deviation would have an
	// enormous z without the floor; it is neither an effect nor a flag.
	series := synth(10, flat20(), hoursAt(8, span(0, 23), scaleKWh(1.12)))
	if got := consumption(series, cfg); len(got) != 0 {
		t.Fatalf("12%% deviation flagged: %+v", got)
	}
}

func flat20() [24]float64 { return flatProfile(20) }

func TestPersistentShiftAndSpikeRules(t *testing.T) {
	cfg := DefaultConfig()
	up := scaleKWh(1.5)
	down := scaleKWh(0.5)

	cases := []struct {
		name      string
		mutate    func(d, h int, r *domain.Reading)
		want      []SignalKind
		wantHours int
		wantSigns []int
	}{
		{
			name:      "six consecutive same-sign hours is persistent",
			mutate:    hoursAt(8, span(2, 7), up),
			want:      []SignalKind{KindPersistentShift},
			wantHours: 6,
			wantSigns: []int{1},
		},
		{
			name:      "five hours is a spike",
			mutate:    hoursAt(8, span(2, 6), up),
			want:      []SignalKind{KindSpike},
			wantHours: 5,
			wantSigns: []int{1},
		},
		{
			name:      "three flagged hours inside a short run is a spike",
			mutate:    hoursAt(8, []int{2, 4, 6}, up),
			want:      []SignalKind{KindSpike},
			wantHours: 5,
			wantSigns: []int{1},
		},
		{
			name: "four up then four down produces two spikes and no persistent shift",
			mutate: func(d, h int, r *domain.Reading) {
				hoursAt(8, span(2, 5), up)(d, h, r)
				hoursAt(8, span(6, 9), down)(d, h, r)
			},
			want:      []SignalKind{KindSpike, KindSpike},
			wantHours: 4,
			wantSigns: []int{1, -1},
		},
		{
			name: "a gap of two unflagged hours merges the run",
			mutate: func(d, h int, r *domain.Reading) {
				hoursAt(8, span(2, 4), up)(d, h, r)
				hoursAt(8, span(7, 9), up)(d, h, r)
			},
			want:      []SignalKind{KindPersistentShift},
			wantHours: 8,
			wantSigns: []int{1},
		},
		{
			name: "a gap of three unflagged hours splits the run",
			mutate: func(d, h int, r *domain.Reading) {
				hoursAt(8, span(2, 4), up)(d, h, r)
				hoursAt(8, span(8, 10), up)(d, h, r)
			},
			want:      []SignalKind{KindSpike, KindSpike},
			wantHours: 3,
			wantSigns: []int{1, 1},
		},
		{
			name:      "a single point of 60 percent or more is a spike",
			mutate:    hoursAt(8, []int{5}, scaleKWh(1.7)),
			want:      []SignalKind{KindSpike},
			wantHours: 1,
			wantSigns: []int{1},
		},
		{
			name:   "a single flagged point below 60 percent is dropped",
			mutate: hoursAt(8, []int{5}, scaleKWh(1.4)),
			want:   nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := consumption(synth(10, flat20(), tc.mutate), cfg)
			gotKinds := kinds(got)
			if len(gotKinds) != len(tc.want) {
				t.Fatalf("signals = %+v, want kinds %v", got, tc.want)
			}
			for i, k := range tc.want {
				if got[i].Kind != k {
					t.Fatalf("signal %d kind = %v, want %v", i, got[i].Kind, k)
				}
				if got[i].Sign != tc.wantSigns[i] {
					t.Fatalf("signal %d sign = %d, want %d", i, got[i].Sign, tc.wantSigns[i])
				}
				if got[i].Role != RoleSeed {
					t.Fatalf("signal %d role = %v, want seed", i, got[i].Role)
				}
			}
			if len(got) > 0 && got[0].Hours != tc.wantHours {
				t.Fatalf("hours = %d, want %d", got[0].Hours, tc.wantHours)
			}
		})
	}
}

func TestEpisodeCarriesNumericEvidence(t *testing.T) {
	cfg := DefaultConfig()
	// 8 hours at +50% over a flat baseline of 20 kWh.
	series := synth(10, flat20(), hoursAt(8, span(2, 9), scaleKWh(1.5)))
	got := consumption(series, cfg)
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	s := got[0]
	near(t, "baseline", s.Baseline, 20)
	near(t, "observed", s.Observed, 30)
	near(t, "delta pct", s.DeltaPct, 50)
	if s.Unit != "kWh" || s.Metric != "consumption_kwh" || s.MeterID != "X-1" {
		t.Fatalf("metadata = %+v", s)
	}
	if s.Flagged != 8 || s.Hours != 8 {
		t.Fatalf("flagged/hours = %d/%d", s.Flagged, s.Hours)
	}
	wantStart := testStart.AddDate(0, 0, 8).Add(2 * 3600e9)
	if !s.Start.Equal(wantStart) {
		t.Fatalf("start = %v, want %v", s.Start, wantStart)
	}
}

func TestInsufficientHistoryYieldsNoConsumptionSignals(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(3, flat20(), hoursAt(2, span(0, 23), scaleKWh(3)))
	if got := consumption(series, cfg); len(got) != 0 {
		t.Fatalf("signals on insufficient history: %+v", got)
	}
}

func TestHourlyDevPctOverrideChangesResults(t *testing.T) {
	series := synth(10, flat20(), hoursAt(8, span(2, 7), scaleKWh(1.47)))
	def := DefaultConfig()
	if got := consumption(series, def); len(got) != 1 || got[0].Kind != KindPersistentShift {
		t.Fatalf("default config: %+v", got)
	}
	strict := DefaultConfig()
	strict.HourlyDevPct = 0.60
	if got := consumption(series, strict); len(got) != 0 {
		t.Fatalf("HourlyDevPct=0.60 still flags a 47%% shift: %+v", got)
	}
}

func dailyVariation(series []domain.Reading, cfg Config) []Signal {
	return DetectDailyVariation(series, BuildBaseline(series, cfg), cfg)
}

func TestDailyVariationFlagsWholeDay(t *testing.T) {
	cfg := DefaultConfig()
	series := synth(10, shapedProfile(), hoursAt(8, span(0, 23), scaleKWh(1.47)))
	got := dailyVariation(series, cfg)
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	s := got[0]
	if s.Kind != KindDailyVariation || s.Role != RoleSecondary || s.Sign != 1 {
		t.Fatalf("signal = %+v", s)
	}
	near(t, "delta pct", s.DeltaPct, 47)
	if s.Unit != "kWh" || s.Hours != 24 {
		t.Fatalf("unit/hours = %q/%d", s.Unit, s.Hours)
	}
	if !s.Start.Equal(testStart.AddDate(0, 0, 8)) {
		t.Fatalf("start = %v", s.Start)
	}
}

func TestDailyVariationIgnoresNormalDaysAndKeepsSign(t *testing.T) {
	cfg := DefaultConfig()
	// +2% and -2% days are silent.
	series := synth(10, shapedProfile(), func(d, h int, r *domain.Reading) {
		hoursAt(8, span(0, 23), scaleKWh(1.02))(d, h, r)
		hoursAt(9, span(0, 23), scaleKWh(0.98))(d, h, r)
	})
	if got := dailyVariation(series, cfg); len(got) != 0 {
		t.Fatalf("normal days flagged: %+v", got)
	}
	// A -40% day is flagged with a negative sign.
	series = synth(10, shapedProfile(), hoursAt(8, span(0, 23), scaleKWh(0.6)))
	got := dailyVariation(series, cfg)
	if len(got) != 1 || got[0].Sign != -1 {
		t.Fatalf("signals = %+v", got)
	}
}

func TestDailyVariationScalesPartialDaysByTheirOwnHours(t *testing.T) {
	cfg := DefaultConfig()
	partial := func(series []domain.Reading) []domain.Reading {
		// keep only hours 0..11 of the last day
		return series[:len(series)-12]
	}
	// Healthy partial day: its energy is about half of DailyKWh but equals the
	// baseline of the hours it contains, so it is silent.
	healthy := partial(synth(10, shapedProfile(), nil))
	if got := dailyVariation(healthy, cfg); len(got) != 0 {
		t.Fatalf("healthy partial day flagged: %+v", got)
	}
	// Tripled partial day is flagged against its own hours.
	tripled := partial(synth(10, shapedProfile(), hoursAt(9, span(0, 11), scaleKWh(3))))
	got := dailyVariation(tripled, cfg)
	if len(got) != 1 {
		t.Fatalf("signals = %+v", got)
	}
	near(t, "delta pct", got[0].DeltaPct, 200)
	if got[0].Hours != 12 {
		t.Fatalf("hours = %d, want 12", got[0].Hours)
	}
}

func TestDailyVariationSkipsInsufficientHistory(t *testing.T) {
	series := synth(3, shapedProfile(), hoursAt(2, span(0, 23), scaleKWh(3)))
	if got := dailyVariation(series, DefaultConfig()); len(got) != 0 {
		t.Fatalf("signals = %+v", got)
	}
}
