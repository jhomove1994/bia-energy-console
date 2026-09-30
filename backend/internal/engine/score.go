package engine

import (
	"fmt"
	"math"
	"sort"

	"energy/internal/domain"
)

// Severity weights and type weights of the priority score.
var (
	severityWeight = map[domain.Severity]float64{
		domain.SeverityHigh:   3,
		domain.SeverityMedium: 2,
		domain.SeverityLow:    1,
	}
	typeWeight = map[domain.AnomalyType]float64{
		domain.TypeReal:        1.0,
		domain.TypeDataQuality: 0.9,
		domain.TypeExplainable: 0.6,
		domain.TypeFalsePos:    0.2,
	}
)

// Confidence labels.
const (
	ConfidenceHigh   = "High"
	ConfidenceMedium = "Medium"
	ConfidenceLow    = "Low"
)

func round(x float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(x*p) / p
}

func clamp(x, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, x)) }

// Band maps a 0-100 severity score to its band.
func Band(score float64, cfg Config) domain.Severity {
	switch {
	case score >= float64(cfg.SevHigh):
		return domain.SeverityHigh
	case score >= float64(cfg.SevMedium):
		return domain.SeverityMedium
	}
	return domain.SeverityLow
}

// SeverityScore applies the type rules to a raw score: it is clamped to
// 0-100, an EXPLAINABLE_ANOMALY is capped just below the HIGH band and a
// FALSE_POSITIVE just below the MEDIUM band.
func SeverityScore(raw float64, t domain.AnomalyType, cfg Config) float64 {
	s := clamp(raw, 0, 100)
	switch t {
	case domain.TypeExplainable:
		s = math.Min(s, float64(cfg.SevHigh-1))
	case domain.TypeFalsePos:
		s = math.Min(s, float64(cfg.SevMedium-1))
	}
	return s
}

// Confidence caps a raw confidence to [0, ConfCap].
func Confidence(raw float64, cfg Config) float64 {
	return clamp(raw, 0, cfg.ConfCap)
}

// ConfidenceLabel names a confidence: High, Medium or Low.
func ConfidenceLabel(conf float64, cfg Config) string {
	switch {
	case conf >= cfg.ConfHigh:
		return ConfidenceHigh
	case conf >= cfg.ConfMedium:
		return ConfidenceMedium
	}
	return ConfidenceLow
}

// Priority combines the severity weight (HIGH 3, MEDIUM 2, LOW 1), the
// confidence and the type weight into a 0-100 score with one decimal.
func Priority(sev domain.Severity, conf float64, t domain.AnomalyType) float64 {
	raw := severityWeight[sev] * conf * typeWeight[t]
	return round(100*raw/severityWeight[domain.SeverityHigh], 1)
}

// Rank sorts anomalies by descending priority score and numbers them from 1.
// Ties are broken by severity score, then meter id, then window start, so the
// order never depends on the input order.
func Rank(as []domain.Anomaly) {
	sort.SliceStable(as, func(i, j int) bool {
		a, b := as[i], as[j]
		switch {
		case a.PriorityScore != b.PriorityScore:
			return a.PriorityScore > b.PriorityScore
		case a.SeverityScore != b.SeverityScore:
			return a.SeverityScore > b.SeverityScore
		case a.MeterID != b.MeterID:
			return a.MeterID < b.MeterID
		}
		return a.WindowStart.Before(b.WindowStart)
	})
	for i := range as {
		as[i].Rank = i + 1
	}
}

// Score builds the anomaly of one classified incident: severity, confidence,
// priority and evidence. Rank, Action and the texts are filled by later
// stages. series and b belong to the incident's meter.
func Score(c Classified, series []domain.Reading, b Baseline, cfg Config) domain.Anomaly {
	a := domain.Anomaly{
		MeterID:     c.MeterID,
		DetectedAt:  c.Start,
		WindowStart: c.Start,
		WindowEnd:   c.End,
		Kind:        c.Kind,
		Type:        c.Type,
		Status:      "open",
	}
	if c.Event != nil {
		e := c.Event.Event
		a.RelatedEvent = &e
	}

	var rawSev, rawConf float64
	if c.Type == domain.TypeDataQuality {
		q := dataQualityStats(c.Incident, series, b)
		rawSev = 50*math.Min(float64(q.jumps)/10, 1) + 30*math.Min(float64(q.hours)/24, 1) + 20*math.Min(float64(q.kinds)/3, 1)
		rawConf = 0.5 + 0.25*math.Min(float64(q.jumps)/10, 1) + 0.15*math.Min(float64(q.kinds)/3, 1)
		if c.Event != nil && c.Event.Event.Type == domain.EventDataQuality {
			rawConf += 0.10
		}
		a.Evidence = q.evidence(c)
	} else {
		w := consumptionStats(c.Incident, series, b, cfg)
		rawSev = 40*math.Min(math.Abs(w.dailyVar), 1) + 20*math.Min(float64(w.hours)/24, 1)
		switch {
		case w.corrob >= 2:
			rawSev += 20
		case w.corrob == 1:
			rawSev += 10
		}
		if c.Type == domain.TypeReal {
			rawSev += 20
		}
		clarity := 1.0
		if c.Event != nil && !c.Event.Explains {
			clarity = 0.5
		}
		rawConf = 0.5 +
			0.2*clamp((math.Abs(w.effect)-cfg.HourlyDevPct)/cfg.HourlyDevPct, 0, 1) +
			0.15*math.Min(float64(w.hours)/24, 1) +
			0.1*float64(w.corrob)/3 +
			0.05*clarity
		a.Evidence = w.evidence(c)
	}

	a.SeverityScore = round(SeverityScore(rawSev, c.Type, cfg), 1)
	a.Severity = Band(a.SeverityScore, cfg)
	a.Confidence = round(Confidence(rawConf, cfg), 3)
	a.PriorityScore = Priority(a.Severity, a.Confidence, c.Type)
	return a
}

// windowHours is the inclusive length of an incident window in hours.
func windowHours(inc Incident) int {
	return int(inc.End.Sub(inc.Start)/hour) + 1
}

// windowStats are the consumption statistics of an incident window.
type windowStats struct {
	hours int
	// Means per hour over the window: observed and hour-of-day baseline.
	obsKWh, baseKWh float64
	obsA, baseA     float64
	obsPF, basePF   float64
	obsV, baseV     float64
	ratio           float64 // median electrical ratio in the window
	hasRatio        bool
	ratioBase       float64

	effect float64
	// dailyVar is the largest variation, by absolute value, of the calendar days
	// that overlap the window; dailyObs and dailyBase belong to that day.
	dailyVar, dailyObs, dailyBase float64

	dA, dPF, dRatio float64
	corrob          int
}

func consumptionStats(inc Incident, series []domain.Reading, b Baseline, cfg Config) windowStats {
	w := windowStats{hours: windowHours(inc), ratioBase: b.RatioMedian}

	var n float64
	var sumObs, sumBase float64
	var ratios []float64
	for _, r := range series {
		if r.TS.Before(inc.Start) || r.TS.After(inc.End) {
			continue
		}
		e := b.At(r.TS)
		n++
		sumObs += r.KWh
		sumBase += e.KWh
		w.obsA += r.CurrentA
		w.baseA += e.CurrentA
		w.obsPF += r.PF
		w.basePF += e.PF
		w.obsV += r.VoltageV
		w.baseV += e.VoltageV
		if q, ok := ElectricalRatio(r); ok {
			ratios = append(ratios, q)
		}
	}
	if n > 0 {
		w.obsKWh, w.baseKWh = sumObs/n, sumBase/n
		w.obsA, w.baseA = w.obsA/n, w.baseA/n
		w.obsPF, w.basePF = w.obsPF/n, w.basePF/n
		w.obsV, w.baseV = w.obsV/n, w.baseV/n
	}
	if sumBase > 0 {
		w.effect = (sumObs - sumBase) / sumBase
	}
	if len(ratios) > 0 {
		w.ratio, w.hasRatio = median(ratios), true
	}

	for _, day := range dayGroups(series) {
		if day[len(day)-1].TS.Before(inc.Start) || day[0].TS.After(inc.End) {
			continue
		}
		var exp, obs float64
		for _, r := range day {
			exp += b.KWh[r.TS.UTC().Hour()].Median
			obs += r.KWh
		}
		if exp <= 0 {
			continue
		}
		if v := (obs - exp) / exp; math.Abs(v) > math.Abs(w.dailyVar) {
			w.dailyVar, w.dailyObs, w.dailyBase = v, obs, exp
		}
	}

	if w.baseA > 0 {
		w.dA = w.obsA/w.baseA - 1
	}
	w.dPF = w.obsPF - w.basePF
	if w.hasRatio && b.RatioMedian > 0 {
		w.dRatio = w.ratio/b.RatioMedian - 1
	}
	if math.Abs(w.dA) >= cfg.CorrCurrentPct {
		w.corrob++
	}
	if math.Abs(w.dPF) >= cfg.CorrPFAbs {
		w.corrob++
	}
	if math.Abs(w.dRatio) >= cfg.CorrRatioPct {
		w.corrob++
	}
	return w
}

func ptr(x float64) *float64 { return &x }

// item builds an evidence item with a baseline; the delta is nil when the
// baseline is zero.
func item(metric, label string, base, obs float64, unit string) domain.Evidence {
	e := domain.Evidence{Metric: metric, Label: label, Baseline: ptr(base), Observed: obs, Unit: unit}
	if base != 0 {
		e.DeltaPct = ptr(pct(obs, base))
	}
	return e
}

// count builds an evidence item without a baseline.
func count(metric, label string, obs float64, unit, detail string) domain.Evidence {
	return domain.Evidence{Metric: metric, Label: label, Observed: obs, Unit: unit, Detail: detail}
}

func (w windowStats) evidence(c Classified) []domain.Evidence {
	ev := []domain.Evidence{
		item("consumption_kwh", "Hourly consumption (window mean)", w.baseKWh, w.obsKWh, "kWh/h"),
		item("daily_kwh", "Daily energy (worst day)", w.dailyBase, w.dailyObs, "kWh"),
		item("current_a", "Current (window mean)", w.baseA, w.obsA, "A"),
		item("power_factor", "Power factor (window mean)", w.basePF, w.obsPF, ""),
		item("voltage_v", "Voltage (window mean)", w.baseV, w.obsV, "V"),
	}
	if w.hasRatio {
		ev = append(ev, item("electrical_ratio", "Electrical ratio (window median)", w.ratioBase, w.ratio, ""))
	}
	ev = append(ev, count("duration_hours", "Duration", float64(w.hours), "h", ""))
	if p, ok := findSignal(c.Incident, KindPatternBreak); ok {
		ev = append(ev, count("hourly_pattern", "Hourly pattern correlation", p.Corr, "corr",
			fmt.Sprintf("corr %.2f, shape deviation %.1f%%", p.Corr, p.Shape*100)))
	}
	return appendEvent(ev, c)
}

// findSignal returns the first seed or attached signal of a kind.
func findSignal(inc Incident, kind SignalKind) (Signal, bool) {
	for _, s := range inc.Seeds {
		if s.Kind == kind {
			return s, true
		}
	}
	for _, s := range inc.Attached {
		if s.Kind == kind {
			return s, true
		}
	}
	return Signal{}, false
}

func appendEvent(ev []domain.Evidence, c Classified) []domain.Evidence {
	if c.Event == nil {
		return ev
	}
	e := c.Event.Event
	id := e.ID
	it := count("event", "Related event", c.Event.Distance.Hours(), "h", fmt.Sprintf("%s: %s", e.Type, e.Description))
	it.EventID = &id
	return append(ev, it)
}

// dqStats are the data-quality statistics of an incident window.
type dqStats struct {
	hours    int
	jumps    int
	byMetric map[string]int
	ratio    bool
	gaps     int
	status   int
	statusV  string
	kinds    int

	series    []domain.Reading
	ratioBase float64
}

func dataQualityStats(inc Incident, series []domain.Reading, b Baseline) dqStats {
	q := dqStats{hours: windowHours(inc), byMetric: map[string]int{}, ratioBase: b.RatioMedian}
	q.series = windowRows(series, inc)
	add := func(s Signal) {
		switch s.Kind {
		case KindDQCluster:
			for m, n := range s.Jumps {
				q.byMetric[m] += n
				q.jumps += n
			}
		case KindJump:
			q.byMetric[s.Metric] += s.Count
			q.jumps += s.Count
		case KindRatioStable, KindRatioUnstable:
			q.ratio = true
		case KindGap:
			q.gaps += s.Count
		case KindStatus:
			q.status += s.Count
			if q.statusV == "" {
				q.statusV = s.Detail
			}
		}
	}
	for _, s := range inc.Seeds {
		add(s)
	}
	for _, s := range inc.Attached {
		add(s)
	}
	for _, m := range []string{metricVoltage, metricCurrent, metricPF} {
		if q.byMetric[m] > 0 {
			q.kinds++
		}
	}
	for _, on := range []bool{q.ratio, q.gaps > 0, q.status > 0} {
		if on {
			q.kinds++
		}
	}
	return q
}

func windowRows(series []domain.Reading, inc Incident) []domain.Reading {
	var out []domain.Reading
	for _, r := range series {
		if !r.TS.Before(inc.Start) && !r.TS.After(inc.End) {
			out = append(out, r)
		}
	}
	return out
}

// minMax returns the smallest and largest value of pick over rows.
func minMax(rows []domain.Reading, pick func(domain.Reading) (float64, bool)) (lo, hi float64, ok bool) {
	for _, r := range rows {
		v, has := pick(r)
		if !has {
			continue
		}
		if !ok {
			lo, hi, ok = v, v, true
			continue
		}
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	return lo, hi, ok
}

func (q dqStats) evidence(c Classified) []domain.Evidence {
	var ev []domain.Evidence
	jumpItem := func(metric, name, unit string, pick func(domain.Reading) (float64, bool)) {
		n := q.byMetric[name]
		if n == 0 {
			return
		}
		detail := ""
		if lo, hi, ok := minMax(q.series, pick); ok {
			detail = fmt.Sprintf("min %.2f %s, max %.2f %s", lo, unit, hi, unit)
		}
		ev = append(ev, count(metric, map[string]string{
			metricVoltage: "Voltage jumps",
			metricCurrent: "Current jumps",
			metricPF:      "Power factor jumps",
		}[name], float64(n), "count", detail))
	}
	jumpItem("voltage_jumps", metricVoltage, "V", func(r domain.Reading) (float64, bool) { return r.VoltageV, true })
	jumpItem("current_jumps", metricCurrent, "A", func(r domain.Reading) (float64, bool) { return r.CurrentA, true })
	jumpItem("pf_jumps", metricPF, "", func(r domain.Reading) (float64, bool) { return r.PF, true })

	if q.ratio {
		pick := func(r domain.Reading) (float64, bool) { return ElectricalRatio(r) }
		detail := ""
		obs := 0.0
		if lo, hi, ok := minMax(q.series, pick); ok {
			detail = fmt.Sprintf("min %.2f, max %.2f", lo, hi)
			var rs []float64
			for _, r := range q.series {
				if v, ok := ElectricalRatio(r); ok {
					rs = append(rs, v)
				}
			}
			obs = median(rs)
		}
		r := item("electrical_ratio", "Electrical ratio (window median)", q.ratioBase, obs, "")
		r.Detail = detail
		ev = append(ev, r)
	}
	ev = append(ev, count("duration_hours", "Duration", float64(q.hours), "h", ""))
	if q.gaps > 0 {
		ev = append(ev, count("gaps", "Missing readings", float64(q.gaps), "count", ""))
	}
	if q.status > 0 {
		ev = append(ev, count("status", "Readings with a non-OK status", float64(q.status), "count", q.statusV))
	}
	return appendEvent(ev, c)
}
