package engine

import (
	"sort"

	"energy/internal/domain"
)

// Input is everything the engine analyzes: the readings of every meter and the
// operational events. Series need not be sorted.
type Input struct {
	Series map[string][]domain.Reading
	Events []domain.Event
}

// Result is the engine output. Anomalies are ordered by rank; Reason and
// RecommendedAction are left to the Explainer. MetersAnalyzed counts the meters
// with enough history to build a baseline.
type Result struct {
	Anomalies      []domain.Anomaly
	MetersAnalyzed int
}

// Detect runs every detector over the series of one meter, which must be sorted
// by timestamp, and returns their signals. A meter without a usable baseline
// yields only the detectors that do not need one.
func Detect(series []domain.Reading, b Baseline, cfg Config) []Signal {
	var out []Signal
	out = append(out, DetectConsumption(series, b, cfg)...)
	out = append(out, DetectDailyVariation(series, b, cfg)...)
	out = append(out, DetectJumps(series, cfg)...)
	out = append(out, DetectRatio(series, b, cfg)...)
	out = append(out, DetectGaps(series)...)
	out = append(out, DetectStatus(series)...)
	out = append(out, DetectPatternBreaks(series, b, cfg)...)
	return out
}

// Analyze chains baseline, detection, correlation, event matching and
// classification, scoring, recommendation and ranking. hook, when not nil, is
// called once as each stage starts, in pipeline order: baseline, detection,
// correlation, events, explanation, recommendation. Stage `readings` belongs to
// the caller. Meters are processed in id order, so the result depends only on
// the input values.
func Analyze(in Input, cfg Config, hook func(domain.Stage)) Result {
	enter := func(s domain.Stage) {
		if hook != nil {
			hook(s)
		}
	}

	ids := make([]string, 0, len(in.Series))
	for id := range in.Series {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	type meter struct {
		series []domain.Reading
		base   Baseline
	}
	meters := map[string]meter{}

	enter(domain.StageBaseline)
	var analyzed []string
	for _, id := range ids {
		series := sortedCopy(in.Series[id])
		b := BuildBaseline(series, cfg)
		if b.InsufficientHistory {
			continue
		}
		meters[id] = meter{series: series, base: b}
		analyzed = append(analyzed, id)
	}

	enter(domain.StageDetection)
	var signals []Signal
	for _, id := range analyzed {
		signals = append(signals, Detect(meters[id].series, meters[id].base, cfg)...)
	}

	enter(domain.StageCorrelation)
	incidents := Correlate(signals, cfg)

	enter(domain.StageEvents)
	classified := make([]Classified, len(incidents))
	for i, inc := range incidents {
		m := meters[inc.MeterID]
		classified[i] = ClassifyIncident(inc, in.Events, m.series, m.base, cfg)
	}

	enter(domain.StageExplanation)
	anomalies := make([]domain.Anomaly, len(classified))
	for i, c := range classified {
		m := meters[c.MeterID]
		anomalies[i] = Score(c, m.series, m.base, cfg)
	}

	enter(domain.StageRecommendation)
	for i, c := range classified {
		anomalies[i].Action = Recommend(c.Type, c.Sign, c.Kind)
	}
	Rank(anomalies)

	return Result{Anomalies: anomalies, MetersAnalyzed: len(analyzed)}
}

// sortedCopy returns the readings ordered by timestamp without touching the
// caller's slice.
func sortedCopy(rs []domain.Reading) []domain.Reading {
	out := append([]domain.Reading(nil), rs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].TS.Before(out[j].TS) })
	return out
}
