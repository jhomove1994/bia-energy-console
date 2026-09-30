package engine

import (
	"testing"

	"energy/internal/domain"
)

// outageWindow is the incident covering hours 192..203 (the first 12 hours of
// day 8) used by the recovery tests.
func outageWindow() Incident {
	return Incident{MeterID: "X-1", Start: hrs(192), End: hrs(203), Sign: -1, Persistent: true}
}

// dropSeries is a flat 20 kWh meter whose first 12 hours of day 8 drop by 60%
// and whose readings are then cut after `until` absolute hours.
func dropSeries(until int) []domain.Reading {
	s := synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
		if d == 8 && h < 12 {
			scaleLoad(r, 0.4)
		}
	})
	return s[:until]
}

func TestRecoveredNeedsTwelveInGateHoursAfterTheWindow(t *testing.T) {
	cfg := DefaultConfig()
	cases := []struct {
		name  string
		until int
		want  bool
	}{
		{"12 in-gate hours", 204 + 12, true},
		{"11 in-gate hours", 204 + 11, false},
		{"many in-gate hours", 384, true},
		{"window ends with the series", 204, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			series := dropSeries(tt.until)
			b := BuildBaseline(series, cfg)
			if got := Recovered(outageWindow(), series, b, cfg); got != tt.want {
				t.Fatalf("Recovered = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecoveryIsBrokenByAFlaggedHourOrAMissingReading(t *testing.T) {
	cfg := DefaultConfig()
	flagged := synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
		if d == 8 && h < 12 {
			scaleLoad(r, 0.4)
		}
		if d == 8 && h == 16 { // 5th hour after the window
			scaleLoad(r, 0.4)
		}
	})
	b := BuildBaseline(flagged, cfg)
	if Recovered(outageWindow(), flagged, b, cfg) {
		t.Fatalf("a flagged hour 5 h after the window must break the 12 h recovery")
	}

	missing := dropAbs(dropSeries(384), 208) // 5th hour after the window is absent
	b = BuildBaseline(missing, cfg)
	if Recovered(outageWindow(), missing, b, cfg) {
		t.Fatalf("a missing reading is not an in-gate hour and must break the recovery")
	}
}

func TestRecoveryUsesConfiguredMinimumAndSkipsInsufficientHistory(t *testing.T) {
	cfg := DefaultConfig()
	series := dropSeries(204 + 6)
	b := BuildBaseline(series, cfg)
	if Recovered(outageWindow(), series, b, cfg) {
		t.Fatalf("6 hours must not recover with the default 12")
	}
	cfg.RecoveryMinHours = 6
	if !Recovered(outageWindow(), series, b, cfg) {
		t.Fatalf("6 hours must recover with RecoveryMinHours=6")
	}
	if Recovered(outageWindow(), series, Baseline{InsufficientHistory: true}, cfg) {
		t.Fatalf("a meter without baseline cannot be judged recovered")
	}
}

// classified matches the events of the incident and classifies it.
func classified(t *testing.T, inc Incident, events []domain.Event, recovered bool) Classified {
	t.Helper()
	return Classify(inc, MatchEvent(inc, events, DefaultConfig()), recovered)
}

func TestClassificationOrder(t *testing.T) {
	down := func(persistent bool) Incident { return consumptionIncident(t, -1, persistent) }
	up := func(persistent bool) Incident { return consumptionIncident(t, 1, persistent) }
	outage := []domain.Event{evAt(1, 200, "SCHEDULED_OUTAGE")}
	opChange := []domain.Event{evAt(1, 200, "OPERATIONAL_CHANGE")}
	unknown := []domain.Event{evAt(1, 200, "UNKNOWN")}
	dq := incidentOf(t, seed(KindDQCluster, 200, 247))
	dqAndShift := func() Incident {
		s := seed(KindPersistentShift, 200, 247)
		s.Sign = 1
		return incidentOf(t, s, seed(KindDQCluster, 210, 230))
	}()

	cases := []struct {
		name      string
		inc       Incident
		events    []domain.Event
		recovered bool
		wantType  domain.AnomalyType
		wantKind  domain.AnomalyKind
		wantEvent bool
	}{
		{"data quality without any event", dq, nil, false, domain.TypeDataQuality, domain.KindDataInconsistency, false},
		{"data quality with its own event keeps the type", dq, []domain.Event{evAt(1, 220, "DATA_QUALITY")}, false, domain.TypeDataQuality, domain.KindDataInconsistency, true},
		{"data quality beats a consumption shift", dqAndShift, opChange, false, domain.TypeDataQuality, domain.KindDataInconsistency, true},
		{"explained and recovered", down(true), outage, true, domain.TypeFalsePos, domain.KindSustainedDecrease, true},
		{"explained and persistent", down(true), outage, false, domain.TypeExplainable, domain.KindLevelShiftDown, true},
		{"operational change up, ongoing", up(true), opChange, false, domain.TypeExplainable, domain.KindLevelShiftUp, true},
		{"operational change down, recovered", down(true), opChange, true, domain.TypeFalsePos, domain.KindSustainedDecrease, true},
		{"outage on an increase does not explain", up(true), outage, false, domain.TypeReal, domain.KindLevelShiftUp, true},
		{"unknown event stays real and is kept as context", up(true), unknown, false, domain.TypeReal, domain.KindLevelShiftUp, true},
		{"no event, recovered increase", up(true), nil, true, domain.TypeReal, domain.KindSustainedIncrease, false},
		{"no event, ongoing decrease", down(true), nil, false, domain.TypeReal, domain.KindLevelShiftDown, false},
		{"spike up", up(false), nil, true, domain.TypeReal, domain.KindConsumptionSpike, false},
		{"spike down", down(false), unknown, true, domain.TypeReal, domain.KindConsumptionDip, true},
		{"outage explains a recovered dip", down(false), outage, true, domain.TypeFalsePos, domain.KindConsumptionDip, true},
		{"operational change does not explain a spike", up(false), opChange, true, domain.TypeReal, domain.KindConsumptionSpike, true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := classified(t, tt.inc, tt.events, tt.recovered)
			if got.Type != tt.wantType || got.Kind != tt.wantKind {
				t.Fatalf("type/kind = %s/%s, want %s/%s", got.Type, got.Kind, tt.wantType, tt.wantKind)
			}
			if (got.Event != nil) != tt.wantEvent {
				t.Fatalf("event kept = %v, want %v", got.Event != nil, tt.wantEvent)
			}
			if got.Recovered != tt.recovered {
				t.Fatalf("Recovered = %v, want %v", got.Recovered, tt.recovered)
			}
		})
	}
}

func TestEveryDataQualitySeedKindClassifiesAsDataQuality(t *testing.T) {
	for _, k := range []SignalKind{KindDQCluster, KindRatioUnstable, KindGap, KindStatus} {
		inc := incidentOf(t, seed(k, 200, 210))
		got := Classify(inc, nil, false)
		if got.Type != domain.TypeDataQuality || got.Kind != domain.KindDataInconsistency {
			t.Fatalf("%s: type/kind = %s/%s", k, got.Type, got.Kind)
		}
	}
	// A stable ratio drift that promoted itself to an incident is data quality too.
	drift := secondary(KindRatioStable, 100, 130)
	drift.Standalone = true
	if got := Classify(incidentOf(t, drift), nil, false); got.Type != domain.TypeDataQuality {
		t.Fatalf("ratio drift type = %s, want DATA_QUALITY", got.Type)
	}
}

func TestOrphanPatternChangeIsRealWithPatternChangeKind(t *testing.T) {
	inc := incidentOf(t, secondary(KindPatternBreak, 192, 215))
	got := Classify(inc, MatchEvent(inc, []domain.Event{evAt(1, 192, "SCHEDULED_OUTAGE")}, DefaultConfig()), true)
	if got.Type != domain.TypeReal || got.Kind != domain.KindPatternChange {
		t.Fatalf("type/kind = %s/%s, want REAL_ANOMALY/PATTERN_CHANGE", got.Type, got.Kind)
	}
}

func TestClassifyIncidentOnSyntheticOutages(t *testing.T) {
	cfg := DefaultConfig()
	outage := []domain.Event{{ID: 1, MeterID: "X-1", TS: hrs(192), Type: domain.EventScheduledOutage}}
	build := func(recovers bool) Classified {
		series := synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
			if d == 8 && (h < 12 || !recovers) || (!recovers && d > 8) {
				scaleLoad(r, 0.4)
			}
		})
		b := BuildBaseline(series, cfg)
		incs := Correlate(allSignals(series, cfg), cfg)
		if len(incs) != 1 {
			t.Fatalf("incidents = %d, want 1", len(incs))
		}
		return ClassifyIncident(incs[0], outage, series, b, cfg)
	}
	rec := build(true)
	if rec.Type != domain.TypeFalsePos || rec.Kind != domain.KindSustainedDecrease || !rec.Recovered || rec.Event == nil || !rec.Event.Explains {
		t.Fatalf("recovered outage = %s/%s recovered=%v event=%+v", rec.Type, rec.Kind, rec.Recovered, rec.Event)
	}
	ongoing := build(false)
	if ongoing.Type != domain.TypeExplainable || ongoing.Kind != domain.KindLevelShiftDown || ongoing.Recovered {
		t.Fatalf("ongoing outage = %s/%s recovered=%v", ongoing.Type, ongoing.Kind, ongoing.Recovered)
	}
	// Same outage without its event is unexplained.
	series := synth(16, flatProfile(20), func(d, h int, r *domain.Reading) {
		if d == 8 && h < 12 {
			scaleLoad(r, 0.4)
		}
	})
	b := BuildBaseline(series, cfg)
	inc := Correlate(allSignals(series, cfg), cfg)[0]
	if got := ClassifyIncident(inc, nil, series, b, cfg); got.Type != domain.TypeReal || got.Kind != domain.KindSustainedDecrease {
		t.Fatalf("no event = %s/%s, want REAL_ANOMALY/SUSTAINED_DECREASE", got.Type, got.Kind)
	}
}
