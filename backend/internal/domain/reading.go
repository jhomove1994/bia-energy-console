package domain

import "time"

// StatusOK is the expected reading status.
const StatusOK = "OK"

// Reading is one hourly measurement for a meter. TS is UTC.
type Reading struct {
	MeterID  string
	TS       time.Time
	KWh      float64
	VoltageV float64
	CurrentA float64
	PF       float64
	Status   string // stored exactly as given
}
