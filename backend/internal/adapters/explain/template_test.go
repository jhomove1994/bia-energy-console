package explain

import (
	"strings"
	"testing"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

var _ app.Explainer = (*Template)(nil)

var t0 = time.Date(2026, 9, 12, 14, 0, 0, 0, time.UTC)

func f(x float64) *float64 { return &x }

func ev(metric string, base, obs, delta float64, unit string) domain.Evidence {
	return domain.Evidence{Metric: metric, Label: metric, Baseline: f(base), Observed: obs, DeltaPct: f(delta), Unit: unit}
}

func consumption() []domain.Evidence {
	return []domain.Evidence{
		ev("consumption_kwh", 20.04, 41.96, 109.38, "kWh/h"),
		ev("daily_kwh", 480.2, 1005.5, 109.4, "kWh"),
		ev("current_a", 30.1, 63.2, 110.0, "A"),
		ev("power_factor", 0.9, 0.7, -22.2, ""),
		ev("voltage_v", 230, 231, 0.4, "V"),
		{Metric: "duration_hours", Label: "Duration", Observed: 58, Unit: "h"},
	}
}

func outage() *domain.Event {
	return &domain.Event{ID: 3, MeterID: "X-1", TS: t0, Type: domain.EventScheduledOutage, Description: "Planned maintenance window"}
}

func anomalies() map[domain.ActionCode]domain.Anomaly {
	base := func(typ domain.AnomalyType, kind domain.AnomalyKind, act domain.ActionCode) domain.Anomaly {
		return domain.Anomaly{
			MeterID: "X-1", Type: typ, Kind: kind, Action: act,
			WindowStart: t0, WindowEnd: t0.Add(57 * time.Hour), Evidence: consumption(),
		}
	}
	m := map[domain.ActionCode]domain.Anomaly{}
	m[domain.ActionInspectLoadIncrease] = base(domain.TypeReal, domain.KindLevelShiftUp, domain.ActionInspectLoadIncrease)
	m[domain.ActionCheckSupplyLoss] = base(domain.TypeReal, domain.KindLevelShiftDown, domain.ActionCheckSupplyLoss)
	m[domain.ActionReviewTransient] = base(domain.TypeReal, domain.KindConsumptionSpike, domain.ActionReviewTransient)
	m[domain.ActionReviewSchedule] = base(domain.TypeReal, domain.KindPatternChange, domain.ActionReviewSchedule)

	dq := base(domain.TypeDataQuality, domain.KindDataInconsistency, domain.ActionOpenDQTicket)
	dq.Evidence = []domain.Evidence{
		{Metric: "voltage_jumps", Observed: 6, Unit: "count", Detail: "mín 200,00 V, máx 251,00 V"},
		{Metric: "current_jumps", Observed: 5, Unit: "count"},
		{Metric: "pf_jumps", Observed: 3, Unit: "count"},
		{Metric: "electrical_ratio", Baseline: f(1.05), Observed: 1.2, Unit: "", Detail: "mín 0,58, máx 4,22"},
		{Metric: "duration_hours", Observed: 48, Unit: "h"},
	}
	m[domain.ActionOpenDQTicket] = dq

	ex := base(domain.TypeExplainable, domain.KindLevelShiftUp, domain.ActionUpdateBaseline)
	ex.RelatedEvent = &domain.Event{ID: 4, Type: domain.EventOperationalChange, Description: "New production line"}
	m[domain.ActionUpdateBaseline] = ex

	fp := base(domain.TypeFalsePos, domain.KindSustainedDecrease, domain.ActionAcknowledgeExpected)
	fp.RelatedEvent = outage()
	m[domain.ActionAcknowledgeExpected] = fp
	return m
}

func TestEveryActionCodeRendersDistinctNonEmptyText(t *testing.T) {
	x := New()
	seen := map[string]domain.ActionCode{}
	for code, a := range anomalies() {
		text := x.Action(a)
		if strings.TrimSpace(text) == "" {
			t.Fatalf("%s: empty action", code)
		}
		if other, dup := seen[text]; dup {
			t.Fatalf("%s and %s render the same action", code, other)
		}
		seen[text] = code
	}
	if len(seen) != 7 {
		t.Fatalf("rendered %d actions, want 7 (one per ActionCode)", len(seen))
	}
}

func TestNoOutputLeaksTemplateArtifacts(t *testing.T) {
	x := New()
	empty := domain.Anomaly{MeterID: "X-1"} // no evidence at all
	for _, typ := range []domain.AnomalyType{domain.TypeReal, domain.TypeDataQuality, domain.TypeExplainable, domain.TypeFalsePos} {
		empty.Type = typ
		for _, a := range append([]domain.Anomaly{empty}, anomalyList()...) {
			for _, out := range []string{x.Reason(a), x.Action(a)} {
				for _, bad := range []string{"<no value>", "{{", "}}", "%!", "<nil>", "NaN"} {
					if strings.Contains(out, bad) {
						t.Fatalf("%s output contains %q: %s", a.Type, bad, out)
					}
				}
			}
		}
	}
}

func anomalyList() []domain.Anomaly {
	var out []domain.Anomaly
	for _, a := range anomalies() {
		out = append(out, a)
	}
	return out
}

func TestReasonsAreNonEmptyAndDifferByType(t *testing.T) {
	x := New()
	seen := map[string]domain.AnomalyType{}
	for _, a := range anomalyList() {
		r := x.Reason(a)
		if strings.TrimSpace(r) == "" {
			t.Fatalf("%s: empty reason", a.Type)
		}
		if a.Kind == domain.KindLevelShiftUp || a.Type == domain.TypeDataQuality || a.Type == domain.TypeFalsePos {
			if other, dup := seen[r]; dup && other != a.Type {
				t.Fatalf("%s and %s render the same reason", a.Type, other)
			}
			seen[r] = a.Type
		}
	}
}

func TestRealReasonCitesEvidenceAndStatesNoEventExplainsIt(t *testing.T) {
	a := anomalies()[domain.ActionInspectLoadIncrease]
	r := New().Reason(a)
	for _, want := range []string{"42,0", "20,0", "+109,4%", "58 h", "12 sept 2026, 14:00 UTC", "1.005,5", "480,2"} {
		if !strings.Contains(r, want) {
			t.Errorf("reason lacks %q: %s", want, r)
		}
	}
	if l := strings.ToLower(r); !strings.Contains(l, "ningún evento conocido explica este cambio") {
		t.Errorf("reason must state that no known event explains the change: %s", r)
	}
}

func TestRealReasonKeepsAnUnrelatedKnownEventAsContext(t *testing.T) {
	a := anomalies()[domain.ActionInspectLoadIncrease]
	a.RelatedEvent = &domain.Event{Type: domain.EventDataQuality, Description: "Sensor recalibrated"}
	r := New().Reason(a)
	for _, want := range []string{"ningún evento explica este cambio", "Calidad de datos", "Sensor recalibrated", "solo contexto"} {
		if !strings.Contains(strings.ToLower(r), strings.ToLower(want)) {
			t.Errorf("reason lacks %q: %s", want, r)
		}
	}
}

func TestRealReasonDoesNotQuoteAnUnknownEvent(t *testing.T) {
	a := anomalies()[domain.ActionInspectLoadIncrease]
	a.RelatedEvent = &domain.Event{Type: domain.EventUnknown, Description: "No operational event reported"}
	r := New().Reason(a)
	if !strings.Contains(r, "Ningún evento conocido explica este cambio.") {
		t.Errorf("reason must say no known event explains the change: %s", r)
	}
	for _, bad := range []string{"No operational event reported", "Desconocido", "UNKNOWN", "contexto"} {
		if strings.Contains(r, bad) {
			t.Errorf("reason must not mention an unknown event (%q): %s", bad, r)
		}
	}
}

func TestReasonsQuoteTheEventAndTheRecovery(t *testing.T) {
	x := New()
	fp := x.Reason(anomalies()[domain.ActionAcknowledgeExpected])
	for _, want := range []string{"Planned maintenance window", "Parada programada", "15 sept 2026, 00:00 UTC"} {
		if !strings.Contains(fp, want) {
			t.Errorf("false positive reason lacks %q: %s", want, fp)
		}
	}
	ex := x.Reason(anomalies()[domain.ActionUpdateBaseline])
	if !strings.Contains(ex, "New production line") || !strings.Contains(ex, "Cambio operativo") {
		t.Errorf("explainable reason lacks the event: %s", ex)
	}
	dq := x.Reason(anomalies()[domain.ActionOpenDQTicket])
	for _, want := range []string{"6 saltos de voltaje", "5 de corriente", "3 de factor de potencia", "mín 0,58, máx 4,22"} {
		if !strings.Contains(dq, want) {
			t.Errorf("data quality reason lacks %q: %s", want, dq)
		}
	}
}

func TestActionsQuoteEventsAndElectricalValues(t *testing.T) {
	x := New()
	up := x.Action(anomalies()[domain.ActionInspectLoadIncrease])
	for _, want := range []string{"X-1", "30,1", "63,2", "0,90", "0,70", "24 h"} {
		if !strings.Contains(up, want) {
			t.Errorf("inspect action lacks %q: %s", want, up)
		}
	}
	if s := x.Action(anomalies()[domain.ActionUpdateBaseline]); !strings.Contains(s, "New production line") {
		t.Errorf("update-baseline action lacks the event: %s", s)
	}
}

func TestRendersAreDeterministic(t *testing.T) {
	a, b := New(), New()
	for _, an := range anomalyList() {
		if a.Reason(an) != a.Reason(an) || a.Reason(an) != b.Reason(an) {
			t.Fatalf("%s: reason differs between renders", an.Type)
		}
		if a.Action(an) != a.Action(an) || a.Action(an) != b.Action(an) {
			t.Fatalf("%s: action differs between renders", an.Type)
		}
	}
}

func TestUnknownActionCodeStillRendersText(t *testing.T) {
	a := domain.Anomaly{MeterID: "X-1", Type: domain.TypeReal, Action: "SOMETHING_NEW"}
	if New().Action(a) == "" {
		t.Fatal("an unknown action code must fall back to a generic text")
	}
}

func TestTextsAreSpanishAndFreeOfEnglishFragments(t *testing.T) {
	x := New()
	english := []string{" the ", "Consumption", "baseline of", "Review ", "Open a", "Dispatch", "No action", " against ", " for ", "explains the"}
	for _, a := range anomalyList() {
		for _, out := range []string{x.Reason(a), x.Action(a)} {
			for _, bad := range english {
				if strings.Contains(out, bad) {
					t.Errorf("%s text still has English %q: %s", a.Type, bad, out)
				}
			}
		}
	}
	generic := x.Action(domain.Anomaly{MeterID: "X-1", Action: "SOMETHING_NEW"})
	if !strings.Contains(generic, "Revisar") {
		t.Errorf("generic action must be Spanish: %s", generic)
	}
}

func TestEventTypeIsShownWithItsSpanishName(t *testing.T) {
	a := anomalies()[domain.ActionInspectLoadIncrease]
	a.RelatedEvent = &domain.Event{Type: domain.EventOperationalChange, Description: "Unclassified entry"}
	if r := New().Reason(a); !strings.Contains(r, "Cambio operativo") || strings.Contains(r, "OPERATIONAL_CHANGE") {
		t.Fatalf("reason = %s", r)
	}
}

func TestPowerFactorRendersWithTwoDecimals(t *testing.T) {
	a := anomalies()[domain.ActionInspectLoadIncrease]
	for i, e := range a.Evidence {
		if e.Metric == "power_factor" {
			a.Evidence[i] = ev("power_factor", 0.94, 0.74, -21.3, "")
		}
	}
	x := New()
	for name, got := range map[string]string{"reason": x.Reason(a), "action": x.Action(a)} {
		for _, want := range []string{"0,94", "0,74"} {
			if !strings.Contains(got, want) {
				t.Errorf("%s lacks %q: %s", name, want, got)
			}
		}
	}
	// Other electrical values keep one decimal.
	if r := x.Reason(a); !strings.Contains(r, "42,0 kWh/h") {
		t.Errorf("consumption must keep one decimal: %s", r)
	}
}
