// Package explain renders anomaly texts from templates. It is deterministic
// and offline: no network, no API key, no language model.
package explain

import (
	"fmt"
	"strings"
	"text/template"
	"time"

	"energy/internal/domain"
)

const timeLayout = "2006-01-02 15:04 UTC"

// Template implements app.Explainer with text/template.
type Template struct {
	reasons map[domain.AnomalyType]*template.Template
	actions map[domain.ActionCode]*template.Template
	generic *template.Template
}

const realReason = `{{if .PatternOnly -}}
The hourly consumption profile of {{.Meter}} broke from its baseline shape for {{.Hours}} h from {{.Start}} (correlation {{.PatternCorr}}). Daily energy was {{.DailyObs}} kWh against {{.DailyBase}} kWh expected.
{{- else -}}
Consumption averaged {{.Obs}} kWh/h against a baseline of {{.Base}} kWh/h ({{.Delta}}%) for {{.Hours}} h from {{.Start}} (worst day {{.DailyObs}} vs {{.DailyBase}} kWh). Current changed {{.ADelta}}%, and power factor moved from {{.PFBase}} to {{.PFObs}}, which corroborates a real load change.
{{- end}} No event explains this change{{if .HasEvent}}: the related {{.EventType}} event ('{{.EventDesc}}') is context only{{end}}.`

const explainableReason = `Consumption averaged {{.Obs}} kWh/h against a baseline of {{.Base}} kWh/h ({{.Delta}}%) for {{.Hours}} h from {{.Start}} and has not returned to the baseline. The related {{.EventType}} event ('{{.EventDesc}}') explains the change, so it is an expected shift rather than a fault.`

const falsePositiveReason = `Consumption averaged {{.Obs}} kWh/h against a baseline of {{.Base}} kWh/h ({{.Delta}}%) for {{.Hours}} h from {{.Start}}, matching the {{.EventType}} event ('{{.EventDesc}}'). Consumption returned to the baseline at {{.Recovery}}, so it is not a real anomaly.`

const dataQualityReason = `The electrical readings of {{.Meter}} were inconsistent for {{.Hours}} h from {{.Start}}: {{.VJumps}} voltage, {{.AJumps}} current and {{.PFJumps}} power factor jumps{{if .RatioDetail}}, and an unstable electrical ratio ({{.RatioDetail}}){{end}}{{if .Gaps}}, {{.Gaps}} missing readings{{end}}{{if .BadStatus}}, {{.BadStatus}} readings with a non-OK status{{end}}. This points to a metering or telemetry fault, not a change in load.{{if .HasEvent}} A related {{.EventType}} event was recorded.{{end}}`

var actionTexts = map[domain.ActionCode]string{
	domain.ActionInspectLoadIncrease: `Dispatch an on-site inspection of {{.Meter}} within 24 h. Check for new, faulty or out-of-schedule loads (current {{.ABase}} to {{.AObs}} A, power factor {{.PFBase}} to {{.PFObs}}) and confirm with the site operator.`,
	domain.ActionCheckSupplyLoss:     `Verify supply continuity and the meter connection for {{.Meter}}. Check for tripped breakers or stopped equipment.`,
	domain.ActionReviewTransient:     `Review load activity on {{.Meter}} during {{.Window}}. Escalate if it recurs.`,
	domain.ActionReviewSchedule:      `Review the operating schedule of {{.Meter}} for {{.Window}}, where the hourly profile broke from its usual shape. Escalate if it persists.`,
	domain.ActionOpenDQTicket:        `Open a metering data-quality ticket for {{.Meter}}. Inspect CT/PT wiring and telemetry, and exclude readings from {{.Window}} from billing and KPIs until validated.`,
	domain.ActionUpdateBaseline:      `Confirm with operations that '{{.EventDesc}}' is permanent and rebaseline {{.Meter}}. No field action needed.`,
	domain.ActionAcknowledgeExpected: `No action required. The deviation matches '{{.EventDesc}}' and consumption recovered at {{.Recovery}}.`,
}

const genericAction = `Review this finding on {{.Meter}} for {{.Window}}.`

// New parses every template once. Templates are constants, so a parse error is
// a programming error.
func New() *Template {
	t := &Template{
		reasons: map[domain.AnomalyType]*template.Template{},
		actions: map[domain.ActionCode]*template.Template{},
	}
	parse := func(name, text string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Parse(text))
	}
	t.reasons[domain.TypeReal] = parse("real", realReason)
	t.reasons[domain.TypeExplainable] = parse("explainable", explainableReason)
	t.reasons[domain.TypeFalsePos] = parse("false-positive", falsePositiveReason)
	t.reasons[domain.TypeDataQuality] = parse("data-quality", dataQualityReason)
	for code, text := range actionTexts {
		t.actions[code] = parse(string(code), text)
	}
	t.generic = parse("generic", genericAction)
	return t
}

// Reason implements app.Explainer.
func (t *Template) Reason(a domain.Anomaly) string {
	tpl, ok := t.reasons[a.Type]
	if !ok {
		tpl = t.generic
	}
	return render(tpl, newView(a))
}

// Action implements app.Explainer.
func (t *Template) Action(a domain.Anomaly) string {
	tpl, ok := t.actions[a.Action]
	if !ok {
		tpl = t.generic
	}
	return render(tpl, newView(a))
}

func render(tpl *template.Template, v view) string {
	var sb strings.Builder
	if err := tpl.Execute(&sb, v); err != nil {
		// The view is a struct of strings, so execution cannot fail; fall back
		// to a plain text rather than an empty explanation.
		return fmt.Sprintf("Review the finding on %s.", v.Meter)
	}
	return strings.TrimSpace(sb.String())
}

// view is the flat, already formatted data the templates read. Every field is
// a string so a missing value renders as an empty string, never as a template
// artifact.
type view struct {
	Meter, Window, Start, Recovery, Hours string

	Obs, Base, Delta        string
	DailyObs, DailyBase     string
	ABase, AObs, ADelta     string
	PFBase, PFObs           string
	PatternOnly             bool
	PatternCorr             string
	VJumps, AJumps, PFJumps string
	RatioDetail             string
	Gaps, BadStatus         string
	HasEvent                bool
	EventType, EventDesc    string
}

func num(x float64) string     { return fmt.Sprintf("%.1f", x) }
func signed(x float64) string  { return fmt.Sprintf("%+.1f", x) }
func whole(x float64) string   { return fmt.Sprintf("%.0f", x) }
func stamp(t time.Time) string { return t.UTC().Format(timeLayout) }

func newView(a domain.Anomaly) view {
	byMetric := map[string]domain.Evidence{}
	for _, e := range a.Evidence {
		byMetric[e.Metric] = e
	}
	v := view{
		Meter:       a.MeterID,
		Start:       stamp(a.WindowStart),
		Window:      stamp(a.WindowStart) + " to " + stamp(a.WindowEnd),
		Recovery:    stamp(a.WindowEnd.Add(time.Hour)),
		Hours:       whole(float64(int(a.WindowEnd.Sub(a.WindowStart)/time.Hour) + 1)),
		PatternOnly: a.Kind == domain.KindPatternChange,
		VJumps:      "0", AJumps: "0", PFJumps: "0",
	}
	if e, ok := byMetric["duration_hours"]; ok {
		v.Hours = whole(e.Observed)
	}

	baseline := func(e domain.Evidence) float64 {
		if e.Baseline == nil {
			return 0
		}
		return *e.Baseline
	}
	delta := func(e domain.Evidence) float64 {
		if e.DeltaPct == nil {
			return 0
		}
		return *e.DeltaPct
	}
	if e, ok := byMetric["consumption_kwh"]; ok {
		v.Obs, v.Base, v.Delta = num(e.Observed), num(baseline(e)), signed(delta(e))
	}
	if e, ok := byMetric["daily_kwh"]; ok {
		v.DailyObs, v.DailyBase = num(e.Observed), num(baseline(e))
	}
	if e, ok := byMetric["current_a"]; ok {
		v.ABase, v.AObs, v.ADelta = num(baseline(e)), num(e.Observed), signed(delta(e))
	}
	if e, ok := byMetric["power_factor"]; ok {
		v.PFBase, v.PFObs = num(baseline(e)), num(e.Observed)
	}
	if e, ok := byMetric["hourly_pattern"]; ok {
		v.PatternCorr = fmt.Sprintf("%.2f", e.Observed)
	}
	if e, ok := byMetric["voltage_jumps"]; ok {
		v.VJumps = whole(e.Observed)
	}
	if e, ok := byMetric["current_jumps"]; ok {
		v.AJumps = whole(e.Observed)
	}
	if e, ok := byMetric["pf_jumps"]; ok {
		v.PFJumps = whole(e.Observed)
	}
	if e, ok := byMetric["electrical_ratio"]; ok && a.Type == domain.TypeDataQuality {
		v.RatioDetail = e.Detail
	}
	if e, ok := byMetric["gaps"]; ok {
		v.Gaps = whole(e.Observed)
	}
	if e, ok := byMetric["status"]; ok {
		v.BadStatus = whole(e.Observed)
	}
	if a.RelatedEvent != nil {
		v.HasEvent = true
		v.EventType = string(a.RelatedEvent.Type)
		v.EventDesc = a.RelatedEvent.Description
	}
	return v
}
