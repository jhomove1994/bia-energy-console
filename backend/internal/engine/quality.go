package engine

import (
	"math"
	"sort"
	"time"

	"energy/internal/domain"
)

// Metric names used by the electrical detectors.
const (
	metricVoltage = "voltage_v"
	metricCurrent = "current_a"
	metricPF      = "power_factor"
)

// jump is one step between two consecutive hourly readings.
type jump struct {
	metric   string
	ts       time.Time // timestamp of the later reading
	sign     int
	previous float64
	current  float64
	meterID  string
}

// findJumps lists the steps of every electrical metric between consecutive
// readings exactly one hour apart. A current step is measured against the
// larger of the two readings so a rise and its reverse have the same size.
func findJumps(series []domain.Reading, cfg Config) []jump {
	var out []jump
	for i := 1; i < len(series); i++ {
		prev, cur := series[i-1], series[i]
		if cur.TS.Sub(prev.TS) != hour {
			continue
		}
		add := func(metric string, a, b float64) {
			out = append(out, jump{metric: metric, ts: cur.TS, sign: sign(b - a), previous: a, current: b, meterID: cur.MeterID})
		}
		if math.Abs(cur.VoltageV-prev.VoltageV) > cfg.VoltJumpV {
			add(metricVoltage, prev.VoltageV, cur.VoltageV)
		}
		if peak := math.Max(prev.CurrentA, cur.CurrentA); peak > 0 && math.Abs(cur.CurrentA-prev.CurrentA)/peak > cfg.CurrentJumpPct {
			add(metricCurrent, prev.CurrentA, cur.CurrentA)
		}
		if math.Abs(cur.PF-prev.PF) >= cfg.PFJump {
			add(metricPF, prev.PF, cur.PF)
		}
	}
	return out
}

// DetectJumps reports electrical instability. A metric oscillates when
// DQWindowHours holds at least DQMinJumps of its jumps with at least
// DQMinReversals sign reversals between consecutive jumps; oscillating jumps
// become data-quality clusters (seeds), merged across metrics when their
// windows overlap or are at most DQWindowHours apart. Every other jump is
// returned as a secondary KindJump signal.
func DetectJumps(series []domain.Reading, cfg Config) []Signal {
	all := findJumps(series, cfg)
	if len(all) == 0 {
		return nil
	}
	window := time.Duration(cfg.DQWindowHours) * hour

	byMetric := map[string][]int{}
	for i, j := range all {
		byMetric[j.metric] = append(byMetric[j.metric], i)
	}

	oscillating := make([]bool, len(all))
	type span struct{ start, end time.Time }
	var spans []span
	for _, idxs := range byMetric {
		marked := markOscillation(all, idxs, window, cfg)
		cur := -1
		var lastTS time.Time
		for _, k := range idxs {
			if !marked[k] {
				continue
			}
			oscillating[k] = true
			ts := all[k].ts
			if cur >= 0 && ts.Sub(lastTS) <= window {
				spans[cur].end = ts
			} else {
				spans = append(spans, span{ts, ts})
				cur = len(spans) - 1
			}
			lastTS = ts
		}
	}

	sort.Slice(spans, func(a, b int) bool { return spans[a].start.Before(spans[b].start) })
	var merged []span
	for _, s := range spans {
		if n := len(merged); n > 0 && s.start.Sub(merged[n-1].end) <= window {
			if s.end.After(merged[n-1].end) {
				merged[n-1].end = s.end
			}
			continue
		}
		merged = append(merged, s)
	}

	var out []Signal
	for _, m := range merged {
		counts := map[string]int{}
		total := 0
		for k, j := range all {
			if oscillating[k] && !j.ts.Before(m.start) && !j.ts.After(m.end) {
				counts[j.metric]++
				total++
			}
		}
		out = append(out, Signal{
			MeterID: all[0].meterID,
			Kind:    KindDQCluster,
			Metric:  "electrical_oscillation",
			Start:   m.start,
			End:     m.end,
			Role:    RoleSeed,
			Hours:   int(m.end.Sub(m.start)/hour) + 1,
			Count:   total,
			Jumps:   counts,
			Unit:    "jumps",
		})
	}
	for k, j := range all {
		if oscillating[k] {
			continue
		}
		out = append(out, Signal{
			MeterID:  j.meterID,
			Kind:     KindJump,
			Metric:   j.metric,
			Start:    j.ts,
			End:      j.ts,
			Sign:     j.sign,
			Role:     RoleSecondary,
			Hours:    1,
			Count:    1,
			Baseline: j.previous,
			Observed: j.current,
			DeltaPct: pct(j.current, j.previous),
			Unit:     metricUnit(j.metric),
		})
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Start.Before(out[b].Start) })
	return out
}

// markOscillation returns, keyed by index into all, the jumps of one metric
// (idxs, chronological) that sit inside a window satisfying the oscillation
// rule.
func markOscillation(all []jump, idxs []int, window time.Duration, cfg Config) map[int]bool {
	marked := map[int]bool{}
	for a := range idxs {
		b := a
		reversals := 0
		for b+1 < len(idxs) && all[idxs[b+1]].ts.Sub(all[idxs[a]].ts) <= window {
			b++
			if all[idxs[b]].sign != all[idxs[b-1]].sign {
				reversals++
			}
		}
		if b-a+1 >= cfg.DQMinJumps && reversals >= cfg.DQMinReversals {
			for k := a; k <= b; k++ {
				marked[idxs[k]] = true
			}
		}
	}
	return marked
}

func metricUnit(metric string) string {
	switch metric {
	case metricVoltage:
		return "V"
	case metricCurrent:
		return "A"
	}
	return ""
}

// ratioHour is one hour whose electrical ratio departs from the baseline.
type ratioHour struct {
	idx   int
	ts    time.Time
	ratio float64
	sign  int
}

// DetectRatio compares the hourly electrical ratio kWh / (V*A*PF/1000) with
// the baseline median. Hours deviating by at least RatioDevPct are deviant.
// Deviant hours of both signs (at least RatioUnstableMinHours of them inside
// DQWindowHours) mean the measurement itself is unreliable: they form
// unstable runs that seed a data-quality incident. The remaining deviant hours
// form same-sign stable runs, which only corroborate a consumption change
// unless they last RatioStandaloneMinHours (Standalone). Rows whose
// denominator is not positive are skipped.
func DetectRatio(series []domain.Reading, b Baseline, cfg Config) []Signal {
	if b.InsufficientHistory || b.RatioMedian <= 0 {
		return nil
	}
	var dev []ratioHour
	for i, r := range series {
		q, ok := ElectricalRatio(r)
		if !ok {
			continue
		}
		d := (q - b.RatioMedian) / b.RatioMedian
		if math.Abs(d) >= cfg.RatioDevPct {
			dev = append(dev, ratioHour{idx: i, ts: r.TS, ratio: q, sign: sign(d)})
		}
	}
	if len(dev) == 0 {
		return nil
	}
	window := time.Duration(cfg.DQWindowHours) * hour

	unstable := make([]bool, len(dev))
	for a := range dev {
		end, pos, neg := a, 0, 0
		for end < len(dev) && dev[end].ts.Sub(dev[a].ts) <= window {
			if dev[end].sign > 0 {
				pos++
			} else {
				neg++
			}
			end++
		}
		if end-a >= cfg.RatioUnstableMinHours && pos > 0 && neg > 0 {
			for k := a; k < end; k++ {
				unstable[k] = true
			}
		}
	}

	var out []Signal
	var group []ratioHour
	flushUnstable := func() {
		if len(group) > 0 {
			out = append(out, ratioSignal(series, b, group, KindRatioUnstable, RoleSeed, 0, cfg))
			group = nil
		}
	}
	for k, h := range dev {
		if !unstable[k] {
			continue
		}
		if n := len(group); n > 0 && h.ts.Sub(group[n-1].ts) > window {
			flushUnstable()
		}
		group = append(group, h)
	}
	flushUnstable()

	flushStable := func() {
		if len(group) > 0 {
			out = append(out, ratioSignal(series, b, group, KindRatioStable, RoleSecondary, group[0].sign, cfg))
			group = nil
		}
	}
	for k, h := range dev {
		if unstable[k] {
			continue
		}
		if n := len(group); n > 0 && (group[n-1].sign != h.sign || gapHours(group[n-1].ts, h.ts) > cfg.MergeGapHours) {
			flushStable()
		}
		group = append(group, h)
	}
	flushStable()

	sort.SliceStable(out, func(a, c int) bool { return out[a].Start.Before(out[c].Start) })
	return out
}

func ratioSignal(series []domain.Reading, b Baseline, group []ratioHour, kind SignalKind, role Role, sgn int, cfg Config) Signal {
	first, last := group[0], group[len(group)-1]
	var sum float64
	for _, h := range group {
		sum += h.ratio
	}
	mean := sum / float64(len(group))
	hours := int(last.ts.Sub(first.ts)/hour) + 1
	return Signal{
		MeterID:    series[first.idx].MeterID,
		Kind:       kind,
		Metric:     "electrical_ratio",
		Start:      first.ts,
		End:        last.ts,
		Sign:       sgn,
		Role:       role,
		Hours:      hours,
		Flagged:    len(group),
		Count:      len(group),
		Baseline:   b.RatioMedian,
		Observed:   mean,
		DeltaPct:   pct(mean, b.RatioMedian),
		Unit:       "ratio",
		Standalone: kind == KindRatioStable && hours >= cfg.RatioStandaloneMinHours,
	}
}
