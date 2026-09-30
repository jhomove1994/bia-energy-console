package engine

import (
	"math"
	"time"

	"energy/internal/domain"
)

// SignalKind identifies the detector that produced a Signal.
type SignalKind string

const (
	KindPersistentShift SignalKind = "persistent_shift"
	KindSpike           SignalKind = "spike"
	KindDailyVariation  SignalKind = "daily_variation"
	KindJump            SignalKind = "jump"
	KindDQCluster       SignalKind = "dq_cluster"
	KindRatioStable     SignalKind = "ratio_stable"
	KindRatioUnstable   SignalKind = "ratio_unstable"
	KindPatternBreak    SignalKind = "pattern_break"
	KindGap             SignalKind = "gap"
	KindStatus          SignalKind = "status"
)

// Role tells the correlation stage how a signal may be used: seeds open
// incidents, secondaries only attach to them.
type Role int

const (
	RoleSeed Role = iota
	RoleSecondary
)

func (r Role) String() string {
	if r == RoleSeed {
		return "seed"
	}
	return "secondary"
}

// Signal is one raw detection with the numeric evidence behind it. Detectors
// never decide what an anomaly is; they only report what they measured.
type Signal struct {
	MeterID string
	Kind    SignalKind
	// Metric is the measured quantity, for example "consumption_kwh".
	Metric string
	// Start and End bound the detection, both inclusive, in UTC.
	Start, End time.Time
	// Sign is +1 for an increase, -1 for a decrease, 0 when not applicable.
	Sign int
	Role Role

	// Hours is the length of the window in hours; Flagged is the number of
	// individually flagged hours inside it; Count is a detector-specific count
	// (jumps, missing timestamps, rows with a bad status).
	Hours, Flagged, Count int

	// Baseline and Observed are the expected and measured values in Unit.
	// DeltaPct is (Observed-Baseline)/Baseline in percent, 0 when Baseline is 0.
	Baseline, Observed, DeltaPct float64
	Unit                         string

	// Jumps counts the jumps per metric inside a DQ cluster.
	Jumps map[string]int
	// Corr and Shape are the pattern-break statistics of a day.
	Corr, Shape float64
	// Standalone marks a stable ratio shift long enough to seed an incident
	// when nothing else covers it.
	Standalone bool
	// Detail carries free text such as the offending status value.
	Detail string
}

const hour = time.Hour

func pct(observed, baseline float64) float64 {
	if baseline == 0 {
		return 0
	}
	return (observed - baseline) / baseline * 100
}

func sign(x float64) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// gapHours is the number of unflagged hours between two timestamps.
func gapHours(from, to time.Time) int {
	return int(to.Sub(from)/hour) - 1
}

// hourFlag is one hour that passed the effect gate and the robust z gate.
type hourFlag struct {
	idx    int
	ts     time.Time
	sign   int
	absDev float64
}

// DetectConsumption flags hours whose kWh departs from the hour-of-day
// baseline, joins same-sign flags into episodes and classifies each episode as
// a persistent shift or a spike. Everything else is discarded. series must be
// sorted by timestamp ascending. A meter with insufficient history yields no
// signals.
func DetectConsumption(series []domain.Reading, b Baseline, cfg Config) []Signal {
	if b.InsufficientHistory {
		return nil
	}
	var out []Signal
	var run []hourFlag
	flush := func() {
		if len(run) == 0 {
			return
		}
		if s, ok := buildEpisode(series, b, cfg, run); ok {
			out = append(out, s)
		}
		run = nil
	}
	for i, r := range series {
		h := r.TS.UTC().Hour()
		med, sigma := b.KWh[h].Median, b.SigmaKWh[h]
		if med <= 0 || sigma <= 0 {
			continue
		}
		delta := r.KWh - med
		dev := delta / med
		if math.Abs(dev) < cfg.HourlyDevPct || math.Abs(delta/sigma) < cfg.RobustZ {
			continue
		}
		f := hourFlag{idx: i, ts: r.TS, sign: sign(dev), absDev: math.Abs(dev)}
		if n := len(run); n > 0 {
			last := run[n-1]
			if last.sign != f.sign || gapHours(last.ts, f.ts) > cfg.MergeGapHours {
				flush()
			}
		}
		run = append(run, f)
	}
	flush()
	return out
}

func buildEpisode(series []domain.Reading, b Baseline, cfg Config, run []hourFlag) (Signal, bool) {
	first, last := run[0], run[len(run)-1]
	hours := int(last.ts.Sub(first.ts)/hour) + 1
	maxDev := 0.0
	for _, f := range run {
		maxDev = math.Max(maxDev, f.absDev)
	}
	kind := KindSpike
	switch {
	case hours >= cfg.PersistMinHours:
		kind = KindPersistentShift
	case len(run) >= cfg.SpikeMinHours || maxDev >= cfg.SpikePointPct:
	default:
		return Signal{}, false
	}

	var base, obs float64
	rows := series[first.idx : last.idx+1]
	for _, r := range rows {
		base += b.KWh[r.TS.UTC().Hour()].Median
		obs += r.KWh
	}
	n := float64(len(rows))
	base, obs = base/n, obs/n
	return Signal{
		MeterID: series[first.idx].MeterID,
		Kind:    kind,
		Metric:  "consumption_kwh",
		Start:   first.ts,
		End:     last.ts,
		Sign:    first.sign,
		Role:    RoleSeed,
		Hours:   hours,
		Flagged: len(run),
		Count:   len(run),

		Baseline: base,
		Observed: obs,
		DeltaPct: pct(obs, base),
		Unit:     "kWh",
	}, true
}

// dayGroups splits a sorted series into runs of readings sharing a UTC date.
func dayGroups(series []domain.Reading) [][]domain.Reading {
	var out [][]domain.Reading
	start := 0
	for i := 1; i <= len(series); i++ {
		if i == len(series) || !sameUTCDay(series[i].TS, series[start].TS) {
			out = append(out, series[start:i])
			start = i
		}
	}
	return out
}

func sameUTCDay(a, b time.Time) bool {
	ay, am, ad := a.UTC().Date()
	by, bm, bd := b.UTC().Date()
	return ay == by && am == bm && ad == bd
}

// DetectDailyVariation compares the energy of each calendar day (UTC) with the
// sum of the hourly medians of the hours that day actually contains, so a
// partial day is scaled by its own hours. Days whose relative deviation
// reaches DailyVarPct are reported as secondary evidence.
func DetectDailyVariation(series []domain.Reading, b Baseline, cfg Config) []Signal {
	if b.InsufficientHistory {
		return nil
	}
	var out []Signal
	for _, day := range dayGroups(series) {
		var expected, observed float64
		for _, r := range day {
			expected += b.KWh[r.TS.UTC().Hour()].Median
			observed += r.KWh
		}
		if expected <= 0 {
			continue
		}
		variation := (observed - expected) / expected
		if math.Abs(variation) < cfg.DailyVarPct {
			continue
		}
		out = append(out, Signal{
			MeterID:  day[0].MeterID,
			Kind:     KindDailyVariation,
			Metric:   "daily_kwh",
			Start:    day[0].TS,
			End:      day[len(day)-1].TS,
			Sign:     sign(variation),
			Role:     RoleSecondary,
			Hours:    len(day),
			Baseline: expected,
			Observed: observed,
			DeltaPct: pct(observed, expected),
			Unit:     "kWh",
		})
	}
	return out
}
