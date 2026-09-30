package domain

// Evidence is one numeric fact supporting an anomaly.
type Evidence struct {
	Metric   string
	Label    string
	Baseline *float64 // nil when not applicable (counts, events)
	Observed float64
	DeltaPct *float64 // nil when Baseline is nil or zero
	Unit     string
	Detail   string
	EventID  *int64 // set on the "event" evidence item
}
