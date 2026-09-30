package engine

import (
	"sort"
	"time"
)

// Incident is one finding window of one meter: the seeds that opened it plus
// the secondary signals attached to it as corroborating evidence.
type Incident struct {
	MeterID string
	// Start and End bound the union of the seed windows, both inclusive. Start
	// is the onset. Attached signals never move either bound.
	Start, End time.Time
	// Sign is the direction of the dominant consumption seed, 0 when the
	// incident has no consumption seed.
	Sign int
	// Persistent is set when a seed is a persistent shift, or a whole-day
	// consumption deviation of at least PersistMinHours.
	Persistent bool
	Seeds      []Signal
	Attached   []Signal
}

func isConsumptionKind(k SignalKind) bool {
	return k == KindPersistentShift || k == KindSpike || k == KindDailyVariation
}

func isDataQualityKind(k SignalKind) bool {
	switch k {
	case KindDQCluster, KindRatioUnstable, KindGap, KindStatus, KindRatioStable:
		return true
	}
	return false
}

// IsConsumption reports whether a seed is a consumption deviation.
func (inc Incident) IsConsumption() bool { return inc.anySeed(isConsumptionKind) }

// IsDataQuality reports whether a seed is evidence of unreliable readings:
// oscillation, unstable ratio, missing readings, a bad status, or a stable
// ratio drift that no consumption seed covers.
func (inc Incident) IsDataQuality() bool { return inc.anySeed(isDataQualityKind) }

// PatternOnly reports whether the incident exists only because of an orphan
// hourly pattern break.
func (inc Incident) PatternOnly() bool {
	return len(inc.Seeds) > 0 && !inc.anySeed(func(k SignalKind) bool { return k != KindPatternBreak })
}

func (inc Incident) anySeed(match func(SignalKind) bool) bool {
	for _, s := range inc.Seeds {
		if match(s.Kind) {
			return true
		}
	}
	return false
}

// Correlate groups the signals of every meter into incidents, exactly one per
// meter per window.
//
// Seeds whose windows overlap or lie at most AttachSlackHours apart merge
// into one incident. Each secondary signal attaches to every incident whose
// window it overlaps within AttachSlackHours; attaching never extends the
// window. Secondaries that attach nowhere are promoted: a daily variation to
// a consumption incident, a standalone stable ratio shift to a data-quality
// incident, a pattern break to a pattern-only incident. Any other orphan (a
// single jump, a short ratio run) is dropped as insufficient evidence.
//
// The result is ordered by meter id, then by onset.
func Correlate(signals []Signal, cfg Config) []Incident {
	byMeter := map[string][]Signal{}
	var meters []string
	for _, s := range signals {
		if _, ok := byMeter[s.MeterID]; !ok {
			meters = append(meters, s.MeterID)
		}
		byMeter[s.MeterID] = append(byMeter[s.MeterID], s)
	}
	sort.Strings(meters)

	var out []Incident
	for _, m := range meters {
		out = append(out, correlateMeter(m, byMeter[m], cfg)...)
	}
	return out
}

func correlateMeter(meter string, signals []Signal, cfg Config) []Incident {
	slack := time.Duration(cfg.AttachSlackHours) * hour

	var seeds, secondaries []Signal
	for _, s := range signals {
		if s.Role == RoleSeed {
			seeds = append(seeds, s)
		} else {
			secondaries = append(secondaries, s)
		}
	}
	sortSignals(seeds)

	incs := make([]Incident, 0, len(seeds))
	for _, s := range seeds {
		incs = append(incs, newIncident(meter, s))
	}
	incs = mergeIncidents(incs, slack, cfg)

	// Orphans are handled class by class, from the strongest evidence to the
	// weakest. Within one class every orphan is promoted before the incidents
	// are merged, so consecutive days become one incident; the next class can
	// then still attach to what was promoted instead of spawning a second
	// incident for the same window.
	for rank := 0; rank <= lastPromotionRank; rank++ {
		var promoted []Incident
		for _, s := range secondaries {
			if promotionRank(s.Kind) != rank || !promotable(s) || attachToAll(incs, s, slack) {
				continue
			}
			promoted = append(promoted, newIncident(meter, s))
		}
		incs = mergeIncidents(append(incs, promoted...), slack, cfg)
	}
	// Whatever cannot open an incident (single jumps, short ratio runs) is
	// evidence when a window covers it and is dropped otherwise.
	for _, s := range secondaries {
		if !promotable(s) {
			attachToAll(incs, s, slack)
		}
	}
	incs = mergeIncidents(incs, slack, cfg)
	return incs
}

// lastPromotionRank is the weakest orphan class that can still be promoted.
const lastPromotionRank = 2

// promotionRank orders orphan secondaries from the strongest evidence to the
// weakest; kinds ranked above lastPromotionRank are never promoted.
func promotionRank(k SignalKind) int {
	switch k {
	case KindDailyVariation:
		return 0
	case KindRatioStable:
		return 1
	case KindPatternBreak:
		return 2
	}
	return 3
}

func promotable(s Signal) bool {
	switch s.Kind {
	case KindDailyVariation, KindPatternBreak:
		return true
	case KindRatioStable:
		return s.Standalone
	}
	return false
}

// attachToAll attaches s to every incident whose window lies within slack of
// it and reports whether it attached anywhere.
func attachToAll(incs []Incident, s Signal, slack time.Duration) bool {
	attached := false
	for i := range incs {
		inc := &incs[i]
		if s.Start.Sub(inc.End) <= slack && inc.Start.Sub(s.End) <= slack {
			inc.Attached = append(inc.Attached, s)
			attached = true
		}
	}
	return attached
}

func newIncident(meter string, seed Signal) Incident {
	return Incident{MeterID: meter, Start: seed.Start, End: seed.End, Seeds: []Signal{seed}}
}

// mergeIncidents merges incidents whose windows overlap or are at most slack
// apart, unions their signals and recomputes the derived traits.
func mergeIncidents(incs []Incident, slack time.Duration, cfg Config) []Incident {
	sort.SliceStable(incs, func(a, b int) bool { return incs[a].Start.Before(incs[b].Start) })
	var out []Incident
	for _, inc := range incs {
		if n := len(out); n > 0 && inc.Start.Sub(out[n-1].End) <= slack {
			last := &out[n-1]
			last.Seeds = append(last.Seeds, inc.Seeds...)
			last.Attached = append(last.Attached, inc.Attached...)
			if inc.End.After(last.End) {
				last.End = inc.End
			}
			continue
		}
		out = append(out, inc)
	}
	for i := range out {
		finalizeIncident(&out[i], cfg)
	}
	return out
}

// finalizeIncident orders the signals deterministically, drops attachments
// that reached the same incident twice and derives Sign and Persistent.
func finalizeIncident(inc *Incident, cfg Config) {
	sortSignals(inc.Seeds)
	sortSignals(inc.Attached)
	inc.Attached = dedupe(inc.Attached)

	inc.Sign, inc.Persistent = 0, false
	dominant := -1
	for i, s := range inc.Seeds {
		if !isConsumptionKind(s.Kind) {
			continue
		}
		if dominant < 0 || s.Hours > inc.Seeds[dominant].Hours {
			dominant = i
		}
		if s.Kind == KindPersistentShift || (s.Kind == KindDailyVariation && s.Hours >= cfg.PersistMinHours) {
			inc.Persistent = true
		}
	}
	if dominant >= 0 {
		inc.Sign = inc.Seeds[dominant].Sign
	}
}

func dedupe(sigs []Signal) []Signal {
	type key struct {
		kind       SignalKind
		metric     string
		start, end time.Time
	}
	seen := map[key]bool{}
	out := sigs[:0]
	for _, s := range sigs {
		k := key{s.Kind, s.Metric, s.Start, s.End}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, s)
	}
	return out
}

func signalLess(a, b Signal) bool {
	if !a.Start.Equal(b.Start) {
		return a.Start.Before(b.Start)
	}
	if !a.End.Equal(b.End) {
		return a.End.Before(b.End)
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Metric < b.Metric
}

func sortSignals(sigs []Signal) {
	sort.SliceStable(sigs, func(a, b int) bool { return signalLess(sigs[a], sigs[b]) })
}
