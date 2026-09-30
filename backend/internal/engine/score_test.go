package engine

import (
	"math"
	"strings"
	"testing"

	"energy/internal/domain"
)

func within(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s = %v, want %v (tol %v)", name, got, want, tol)
	}
}

func TestBandBoundaries(t *testing.T) {
	cfg := DefaultConfig()
	cases := []struct {
		score float64
		want  domain.Severity
	}{
		{70, domain.SeverityHigh},
		{69, domain.SeverityMedium},
		{40, domain.SeverityMedium},
		{39, domain.SeverityLow},
	}
	for _, tt := range cases {
		if got := Band(tt.score, cfg); got != tt.want {
			t.Errorf("Band(%v) = %s, want %s", tt.score, got, tt.want)
		}
	}
}

func TestSeverityCapsFollowTheType(t *testing.T) {
	cfg := DefaultConfig()
	cases := []struct {
		name     string
		raw      float64
		typ      domain.AnomalyType
		want     float64
		wantBand domain.Severity
	}{
		{"explainable 85 capped to medium", 85, domain.TypeExplainable, 69, domain.SeverityMedium},
		{"explainable below the cap untouched", 48.8, domain.TypeExplainable, 48.8, domain.SeverityMedium},
		{"false positive 55 forced low", 55, domain.TypeFalsePos, 39, domain.SeverityLow},
		{"false positive already low", 12, domain.TypeFalsePos, 12, domain.SeverityLow},
		{"real is never capped", 100, domain.TypeReal, 100, domain.SeverityHigh},
		{"data quality is never capped", 90, domain.TypeDataQuality, 90, domain.SeverityHigh},
		{"score above 100 is clamped", 130, domain.TypeReal, 100, domain.SeverityHigh},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := SeverityScore(tt.raw, tt.typ, cfg)
			within(t, "score", got, tt.want, 1e-9)
			if b := Band(got, cfg); b != tt.wantBand {
				t.Fatalf("band = %s, want %s", b, tt.wantBand)
			}
		})
	}
}

func TestSeverityCapsFollowTheConfiguredBands(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SevHigh, cfg.SevMedium = 80, 50
	within(t, "explainable", SeverityScore(95, domain.TypeExplainable, cfg), 79, 1e-9)
	within(t, "false positive", SeverityScore(95, domain.TypeFalsePos, cfg), 49, 1e-9)
}

func TestConfidenceIsCappedAndLabeled(t *testing.T) {
	cfg := DefaultConfig()
	within(t, "cap", Confidence(1.2, cfg), 0.98, 1e-9)
	within(t, "below the cap", Confidence(0.846, cfg), 0.846, 1e-9)
	within(t, "floor", Confidence(-0.2, cfg), 0, 1e-9)

	cases := []struct {
		conf float64
		want string
	}{
		{0.98, "High"}, {0.85, "High"}, {0.84, "Medium"}, {0.60, "Medium"}, {0.59, "Low"},
	}
	for _, tt := range cases {
		if got := ConfidenceLabel(tt.conf, cfg); got != tt.want {
			t.Errorf("ConfidenceLabel(%v) = %s, want %s", tt.conf, got, tt.want)
		}
	}
}

func TestPriorityCombinesSeverityConfidenceAndType(t *testing.T) {
	within(t, "real high", Priority(domain.SeverityHigh, 0.975, domain.TypeReal), 97.5, 1e-9)
	within(t, "dq high", Priority(domain.SeverityHigh, 0.98, domain.TypeDataQuality), 88.2, 1e-9)
	within(t, "explainable medium", Priority(domain.SeverityMedium, 0.846, domain.TypeExplainable), 33.8, 1e-9)
	within(t, "false positive low", Priority(domain.SeverityLow, 0.858, domain.TypeFalsePos), 5.7, 1e-9)

	real := Priority(domain.SeverityHigh, 0.9, domain.TypeReal)
	dq := Priority(domain.SeverityHigh, 0.9, domain.TypeDataQuality)
	if real <= dq {
		t.Fatalf("REAL (%v) must beat DATA_QUALITY (%v) on equal severity and confidence", real, dq)
	}
}

func TestRankIsDescendingAndDeterministic(t *testing.T) {
	mk := func(meter string, prio, sev float64) domain.Anomaly {
		return domain.Anomaly{MeterID: meter, PriorityScore: prio, SeverityScore: sev}
	}
	in := []domain.Anomaly{
		mk("B-2", 50, 40),
		mk("A-1", 50, 40), // ties with B-2 on both scores: meter id decides
		mk("C-3", 50, 60), // same priority, higher severity wins
		mk("D-4", 90, 10),
		mk("E-5", 5, 5),
	}
	want := []string{"D-4", "C-3", "A-1", "B-2", "E-5"}

	perms := [][]int{{0, 1, 2, 3, 4}, {4, 3, 2, 1, 0}, {2, 0, 4, 1, 3}}
	for _, p := range perms {
		as := make([]domain.Anomaly, len(in))
		for i, k := range p {
			as[i] = in[k]
		}
		Rank(as)
		for i, a := range as {
			if a.MeterID != want[i] || a.Rank != i+1 {
				t.Fatalf("perm %v: position %d = %s rank %d, want %s rank %d", p, i, a.MeterID, a.Rank, want[i], i+1)
			}
		}
	}
}

// scoredIncident is a persistent increase over days 8 and 9 (hours 192..239).
func scoredIncident() Incident {
	return Incident{
		MeterID:    "X-1",
		Start:      hrs(192),
		End:        hrs(239),
		Sign:       1,
		Persistent: true,
		Seeds:      []Signal{{Kind: KindPersistentShift, Role: RoleSeed, Start: hrs(192), End: hrs(239)}},
	}
}

// doubledSeries is a flat 20 kWh meter whose load doubles on days 8 and 9.
func doubledSeries() []domain.Reading {
	return synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
		if d == 8 || d == 9 {
			scaleLoad(r, 2)
		}
	})
}

func evidenceByMetric(a domain.Anomaly) map[string]domain.Evidence {
	m := map[string]domain.Evidence{}
	for _, e := range a.Evidence {
		m[e.Metric] = e
	}
	return m
}

func TestScoreRealConsumptionIncident(t *testing.T) {
	cfg := DefaultConfig()
	series := doubledSeries()
	b := BuildBaseline(series, cfg)

	c := Classify(scoredIncident(), nil, false)
	a := Score(c, series, b, cfg)

	// 40*1 (daily +100%) + 20*1 (48 h) + 10 (only the current corroborates) + 20 (REAL)
	within(t, "severity score", a.SeverityScore, 90, 0.05)
	if a.Severity != domain.SeverityHigh {
		t.Fatalf("severity = %s, want HIGH", a.Severity)
	}
	// 0.5 + 0.2 + 0.15 + 0.1*1/3 + 0.05 (no event)
	within(t, "confidence", a.Confidence, 0.933, 0.001)
	within(t, "priority", a.PriorityScore, 100*3*a.Confidence*1.0/3, 0.05)

	if a.Type != domain.TypeReal || a.Kind != domain.KindLevelShiftUp {
		t.Fatalf("type/kind = %s/%s", a.Type, a.Kind)
	}
	if !a.WindowStart.Equal(hrs(192)) || !a.DetectedAt.Equal(a.WindowStart) || !a.WindowEnd.Equal(hrs(239)) {
		t.Fatalf("window = %v..%v detected %v", a.WindowStart, a.WindowEnd, a.DetectedAt)
	}
	if a.MeterID != "X-1" || a.Status != "open" || a.RelatedEvent != nil {
		t.Fatalf("meter/status/event = %s/%s/%v", a.MeterID, a.Status, a.RelatedEvent)
	}
}

func TestScoreEvidenceItemsCarryAllSixFields(t *testing.T) {
	cfg := DefaultConfig()
	series := doubledSeries()
	b := BuildBaseline(series, cfg)
	a := Score(Classify(scoredIncident(), nil, false), series, b, cfg)
	ev := evidenceByMetric(a)

	for _, m := range []string{"consumption_kwh", "daily_kwh", "current_a", "power_factor", "voltage_v", "electrical_ratio", "duration_hours"} {
		if _, ok := ev[m]; !ok {
			t.Fatalf("missing evidence %q; got %+v", m, a.Evidence)
		}
	}
	if _, ok := ev["event"]; ok {
		t.Fatal("no event matched, so no event evidence is expected")
	}
	for _, m := range []string{"consumption_kwh", "current_a"} {
		e := ev[m]
		if e.Label == "" || e.Unit == "" || e.Baseline == nil || e.DeltaPct == nil {
			t.Fatalf("%s: incomplete evidence %+v", m, e)
		}
	}
	c := ev["consumption_kwh"]
	within(t, "consumption baseline", *c.Baseline, 20, 1e-6)
	within(t, "consumption observed", c.Observed, 40, 1e-6)
	within(t, "consumption delta", *c.DeltaPct, 100, 1e-6)
	if c.Unit != "kWh/h" {
		t.Fatalf("consumption unit = %q", c.Unit)
	}
	d := ev["daily_kwh"]
	within(t, "daily baseline", *d.Baseline, 480, 1e-6)
	within(t, "daily observed", d.Observed, 960, 1e-6)
	within(t, "current delta", *ev["current_a"].DeltaPct, 100, 1e-6)
	within(t, "duration", ev["duration_hours"].Observed, 48, 1e-9)
	if ev["duration_hours"].Baseline != nil || ev["duration_hours"].DeltaPct != nil {
		t.Fatal("a duration has no baseline or delta")
	}
}

func TestScoreKeepsTheRelatedEventAsEvidenceAndContext(t *testing.T) {
	cfg := DefaultConfig()
	series := doubledSeries()
	b := BuildBaseline(series, cfg)
	ev := domain.Event{ID: 7, MeterID: "X-1", TS: hrs(192), Type: domain.EventOperationalChange, Description: "new line"}
	match := &EventMatch{Event: ev, Power: 0.9, Explains: true}

	a := Score(Classify(scoredIncident(), match, false), series, b, cfg)
	if a.RelatedEvent == nil || a.RelatedEvent.ID != 7 {
		t.Fatalf("RelatedEvent = %+v", a.RelatedEvent)
	}
	e := evidenceByMetric(a)["event"]
	if e.EventID == nil || *e.EventID != 7 || e.Detail != "Cambio operativo: new line" {
		t.Fatalf("event evidence = %+v", e)
	}
	// Explained by a known event: clarity 1, so 0.5+0.2+0.15+0.033+0.05.
	within(t, "confidence", a.Confidence, 0.933, 0.001)

	unknown := &EventMatch{Event: domain.Event{ID: 8, MeterID: "X-1", TS: hrs(192), Type: domain.EventUnknown}}
	u := Score(Classify(scoredIncident(), unknown, false), series, b, cfg)
	// Unrecognized event: clarity 0.5, so 0.025 instead of 0.05.
	within(t, "unknown-event confidence", u.Confidence, 0.908, 0.001)
	if u.Type != domain.TypeReal || u.RelatedEvent == nil {
		t.Fatalf("type %s, event %+v: an UNKNOWN event stays context only", u.Type, u.RelatedEvent)
	}
}

func TestScoreCapsExplainableAndForcesFalsePositiveLow(t *testing.T) {
	cfg := DefaultConfig()
	series := doubledSeries()
	b := BuildBaseline(series, cfg)
	op := &EventMatch{Event: domain.Event{ID: 1, Type: domain.EventOperationalChange}, Power: 0.9, Explains: true}

	// Raw 70 would be HIGH: the operational change caps it at 69.
	a := Score(Classify(scoredIncident(), op, false), series, b, cfg)
	if a.Type != domain.TypeExplainable || a.Severity != domain.SeverityMedium {
		t.Fatalf("type/severity = %s/%s, want EXPLAINABLE/MEDIUM", a.Type, a.Severity)
	}
	within(t, "explainable score", a.SeverityScore, 69, 1e-9)

	out := &EventMatch{Event: domain.Event{ID: 2, Type: domain.EventScheduledOutage}, Power: 1, Explains: true}
	inc := scoredIncident()
	inc.Sign = -1
	f := Score(Classify(inc, out, true), series, b, cfg)
	if f.Type != domain.TypeFalsePos || f.Severity != domain.SeverityLow {
		t.Fatalf("type/severity = %s/%s, want FALSE_POSITIVE/LOW", f.Type, f.Severity)
	}
	if f.SeverityScore > 39 {
		t.Fatalf("false positive score = %v, want <= 39", f.SeverityScore)
	}
}

// dqSeries toggles voltage and current every other hour of days 8 and 9.
func dqSeries() []domain.Reading {
	return synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
		if (d == 8 || d == 9) && h%2 == 0 {
			r.VoltageV += 40
			r.CurrentA *= 3
		}
	})
}

func dqIncident() Incident {
	return Incident{
		MeterID: "X-1",
		Start:   hrs(192),
		End:     hrs(239),
		Seeds: []Signal{
			{Kind: KindDQCluster, Role: RoleSeed, Start: hrs(192), End: hrs(239), Count: 12,
				Jumps: map[string]int{metricVoltage: 6, metricCurrent: 6}},
			{Kind: KindRatioUnstable, Role: RoleSeed, Start: hrs(192), End: hrs(239), Count: 20},
		},
		Attached: []Signal{
			{Kind: KindJump, Role: RoleSecondary, Metric: metricPF, Start: hrs(200), End: hrs(200), Count: 1},
		},
	}
}

func TestScoreDataQualityIncident(t *testing.T) {
	cfg := DefaultConfig()
	series := dqSeries()
	b := BuildBaseline(series, cfg)

	a := Score(Classify(dqIncident(), nil, false), series, b, cfg)
	if a.Type != domain.TypeDataQuality || a.Kind != domain.KindDataInconsistency {
		t.Fatalf("type/kind = %s/%s", a.Type, a.Kind)
	}
	// 13 jumps (cap 10) -> 50, 48 h -> 30, kinds voltage/current/pf/ratio = 4 -> 20.
	within(t, "severity", a.SeverityScore, 100, 1e-9)
	if a.Severity != domain.SeverityHigh {
		t.Fatalf("severity = %s", a.Severity)
	}
	// 0.5 + 0.25 + 0.15 = 0.90, no DATA_QUALITY event.
	within(t, "confidence", a.Confidence, 0.90, 1e-9)

	ev := evidenceByMetric(a)
	for _, m := range []string{"voltage_jumps", "current_jumps", "pf_jumps", "electrical_ratio", "duration_hours"} {
		if _, ok := ev[m]; !ok {
			t.Fatalf("missing evidence %q; got %+v", m, a.Evidence)
		}
	}
	within(t, "voltage jumps", ev["voltage_jumps"].Observed, 6, 1e-9)
	within(t, "pf jumps", ev["pf_jumps"].Observed, 1, 1e-9)
	if !strings.HasPrefix(ev["voltage_jumps"].Detail, "mín ") || !strings.Contains(ev["voltage_jumps"].Detail, " V, máx ") || !strings.HasPrefix(ev["electrical_ratio"].Detail, "mín ") {
		t.Fatalf("min/max details missing or not Spanish: %+v", a.Evidence)
	}
	if _, ok := ev["gaps"]; ok {
		t.Fatal("no gap signal, so no gaps evidence")
	}
}

func TestDataQualityEventRaisesConfidenceButNotType(t *testing.T) {
	cfg := DefaultConfig()
	series := dqSeries()
	b := BuildBaseline(series, cfg)
	match := &EventMatch{Event: domain.Event{ID: 9, MeterID: "X-1", Type: domain.EventDataQuality, Description: "sensor check"}}

	a := Score(Classify(dqIncident(), match, false), series, b, cfg)
	if a.Type != domain.TypeDataQuality {
		t.Fatalf("type = %s", a.Type)
	}
	// 0.9 + 0.10 = 1.0 -> capped.
	within(t, "confidence", a.Confidence, 0.98, 1e-9)
	if a.RelatedEvent == nil || evidenceByMetric(a)["event"].EventID == nil {
		t.Fatal("the DATA_QUALITY event must appear as related event and evidence")
	}
}

func TestDataQualityEvidenceReportsGapsAndStatus(t *testing.T) {
	cfg := DefaultConfig()
	series := dqSeries()
	b := BuildBaseline(series, cfg)
	inc := Incident{
		MeterID: "X-1", Start: hrs(192), End: hrs(215),
		Seeds: []Signal{
			{Kind: KindGap, Role: RoleSeed, Start: hrs(192), End: hrs(195), Count: 4},
			{Kind: KindStatus, Role: RoleSeed, Start: hrs(200), End: hrs(205), Count: 6, Detail: "SENSOR_FAULT"},
		},
	}
	a := Score(Classify(inc, nil, false), series, b, cfg)
	ev := evidenceByMetric(a)
	within(t, "gaps", ev["gaps"].Observed, 4, 1e-9)
	within(t, "status", ev["status"].Observed, 6, 1e-9)
	if ev["status"].Detail != "SENSOR_FAULT" {
		t.Fatalf("status detail = %q", ev["status"].Detail)
	}
	// 0 jumps, 24 h -> 30, two kinds -> 13.33
	within(t, "severity", a.SeverityScore, 43.3, 0.05)
}

func TestScoreAddsPatternEvidenceWhenABreakIsAttached(t *testing.T) {
	cfg := DefaultConfig()
	series := doubledSeries()
	b := BuildBaseline(series, cfg)
	inc := scoredIncident()
	inc.Attached = []Signal{{Kind: KindPatternBreak, Role: RoleSecondary, Start: hrs(192), End: hrs(215), Corr: 0.53, Shape: 0.368, Observed: 0.53, Unit: "correlation"}}

	e, ok := evidenceByMetric(Score(Classify(inc, nil, false), series, b, cfg))["hourly_pattern"]
	if !ok {
		t.Fatal("missing hourly_pattern evidence")
	}
	within(t, "corr", e.Observed, 0.53, 1e-9)
	if e.Detail != "correlación 0,53, desviación de forma 36,8%" {
		t.Fatalf("detail = %q", e.Detail)
	}
}
