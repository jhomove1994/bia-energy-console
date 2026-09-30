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

// Template implements app.Explainer with text/template.
type Template struct {
	reasons map[domain.AnomalyType]*template.Template
	actions map[domain.ActionCode]*template.Template
	generic *template.Template
}

const realReason = `{{if .PatternOnly -}}
El perfil horario de consumo de {{.Meter}} se desvió de su forma base durante {{.Hours}} h desde el {{.Start}} (correlación {{.PatternCorr}}). La energía diaria fue de {{.DailyObs}} kWh frente a {{.DailyBase}} kWh esperados.
{{- else -}}
El consumo promedió {{.Obs}} kWh/h frente a un baseline de {{.Base}} kWh/h ({{.Delta}}%) durante {{.Hours}} h desde el {{.Start}} (peor día: {{.DailyObs}} frente a {{.DailyBase}} kWh). La corriente varió {{.ADelta}}% y el factor de potencia pasó de {{.PFBase}} a {{.PFObs}}, lo que corrobora un cambio real de carga.
{{- end}}{{if .ContextEvent}} Ningún evento explica este cambio: el evento relacionado de tipo {{.EventType}} («{{.EventDesc}}») es solo contexto.{{else}} Ningún evento conocido explica este cambio.{{end}}`

const explainableReason = `El consumo promedió {{.Obs}} kWh/h frente a un baseline de {{.Base}} kWh/h ({{.Delta}}%) durante {{.Hours}} h desde el {{.Start}} y no ha vuelto al baseline. El evento relacionado de tipo {{.EventType}} («{{.EventDesc}}») explica el cambio, por lo que se trata de una variación esperada y no de una falla.`

const falsePositiveReason = `El consumo promedió {{.Obs}} kWh/h frente a un baseline de {{.Base}} kWh/h ({{.Delta}}%) durante {{.Hours}} h desde el {{.Start}}, en coincidencia con el evento de tipo {{.EventType}} («{{.EventDesc}}»). El consumo volvió al baseline el {{.Recovery}}, por lo que no es una anomalía real.`

const dataQualityReason = `Las lecturas eléctricas de {{.Meter}} fueron inconsistentes durante {{.Hours}} h desde el {{.Start}}: {{.VJumps}} saltos de voltaje, {{.AJumps}} de corriente y {{.PFJumps}} de factor de potencia{{if .RatioDetail}}, y una relación eléctrica inestable ({{.RatioDetail}}){{end}}{{if .Gaps}}, {{.Gaps}} lecturas faltantes{{end}}{{if .BadStatus}}, {{.BadStatus}} lecturas con estado distinto de OK{{end}}. Esto apunta a una falla de medición o telemetría, no a un cambio de carga.{{if .HasEvent}} Se registró un evento relacionado de tipo {{.EventType}}.{{end}}`

var actionTexts = map[domain.ActionCode]string{
	domain.ActionInspectLoadIncrease: `Enviar una inspección en sitio a {{.Meter}} en un plazo de 24 h. Verificar cargas nuevas, defectuosas o fuera de horario (corriente de {{.ABase}} a {{.AObs}} A, factor de potencia de {{.PFBase}} a {{.PFObs}}) y confirmar con el operador del sitio.`,
	domain.ActionCheckSupplyLoss:     `Verificar la continuidad del suministro y la conexión del medidor de {{.Meter}}. Revisar si hay interruptores disparados o equipos detenidos.`,
	domain.ActionReviewTransient:     `Revisar la actividad de carga de {{.Meter}} en el periodo {{.Window}}. Escalar si se repite.`,
	domain.ActionReviewSchedule:      `Revisar el horario de operación de {{.Meter}} en el periodo {{.Window}}, en el que el perfil horario se desvió de su forma habitual. Escalar si persiste.`,
	domain.ActionOpenDQTicket:        `Abrir un ticket de calidad de datos de medición para {{.Meter}}. Inspeccionar el cableado CT/PT y la telemetría, y excluir de la facturación y de los KPI las lecturas {{.Window}} hasta su validación.`,
	domain.ActionUpdateBaseline:      `Confirmar con operaciones que «{{.EventDesc}}» es permanente y recalcular el baseline de {{.Meter}}. No se requiere acción en campo.`,
	domain.ActionAcknowledgeExpected: `No se requiere acción. La desviación coincide con «{{.EventDesc}}» y el consumo se recuperó el {{.Recovery}}.`,
}

const genericAction = `Revisar este hallazgo de {{.Meter}} en el periodo {{.Window}}.`

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
		return fmt.Sprintf("Revisar el hallazgo de %s.", v.Meter)
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
	// ContextEvent is a related event of a known type; an unknown entry is not worth quoting.
	ContextEvent         bool
	EventType, EventDesc string
}

func num(x float64) string     { return domain.FormatNumber(x, 1) }
func ratio(x float64) string   { return domain.FormatNumber(x, 2) }
func signed(x float64) string  { return domain.FormatSigned(x, 1) }
func whole(x float64) string   { return domain.FormatNumber(x, 0) }
func stamp(t time.Time) string { return domain.FormatStamp(t) }

func newView(a domain.Anomaly) view {
	byMetric := map[string]domain.Evidence{}
	for _, e := range a.Evidence {
		byMetric[e.Metric] = e
	}
	v := view{
		Meter:       a.MeterID,
		Start:       stamp(a.WindowStart),
		Window:      "del " + stamp(a.WindowStart) + " al " + stamp(a.WindowEnd),
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
		v.PFBase, v.PFObs = ratio(baseline(e)), ratio(e.Observed)
	}
	if e, ok := byMetric["hourly_pattern"]; ok {
		v.PatternCorr = ratio(e.Observed)
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
		v.EventType = domain.EventTypeName(a.RelatedEvent.Type)
		v.EventDesc = a.RelatedEvent.Description
		v.ContextEvent = domain.EventTypeName(a.RelatedEvent.Type) != domain.EventTypeName(domain.EventUnknown)
	}
	return v
}
