package domain

import (
	"strings"
	"time"
)

// EventType is the normalized operational event type.
type EventType string

const (
	EventScheduledOutage   EventType = "SCHEDULED_OUTAGE"
	EventOperationalChange EventType = "OPERATIONAL_CHANGE"
	EventDataQuality       EventType = "DATA_QUALITY"
	EventUnknown           EventType = "UNKNOWN"
)

// Event is an operational event reported for a meter.
type Event struct {
	ID          int64
	MeterID     string
	TS          time.Time
	Type        EventType
	Description string
}

// NormalizeEventType trims and uppercases raw, maps maintenance-like types to
// SCHEDULED_OUTAGE and any unrecognized value to UNKNOWN.
func NormalizeEventType(raw string) EventType {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "SCHEDULED_OUTAGE", "MAINTENANCE", "SCHEDULED_MAINTENANCE", "PLANNED_OUTAGE":
		return EventScheduledOutage
	case "OPERATIONAL_CHANGE":
		return EventOperationalChange
	case "DATA_QUALITY":
		return EventDataQuality
	default:
		return EventUnknown
	}
}
