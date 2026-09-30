package engine

import (
	"time"

	"energy/internal/domain"
)

// Explanatory power of an event type when its direction fits the incident.
const (
	powerScheduledOutage   = 1.0
	powerOperationalChange = 0.9
)

// EventMatch is the event related to an incident.
type EventMatch struct {
	// Event carries the normalized type.
	Event domain.Event
	// Power is how strongly the event explains the incident, 0 when its type
	// or direction does not fit. It is never negative.
	Power float64
	// Explains is set when Power reaches ExplainPowerMin.
	Explains bool
	// Distance is the absolute time between the event and the onset.
	Distance time.Duration
}

// explanatoryPower is direction aware: a scheduled outage explains decreases
// only, an operational change explains persistent shifts of either sign, and
// data-quality, unknown and unrecognized events explain nothing. Only
// consumption incidents can be explained.
func explanatoryPower(t domain.EventType, inc Incident) float64 {
	if !inc.IsConsumption() {
		return 0
	}
	switch t {
	case domain.EventScheduledOutage:
		if inc.Sign < 0 {
			return powerScheduledOutage
		}
	case domain.EventOperationalChange:
		if inc.Persistent {
			return powerOperationalChange
		}
	}
	return 0
}

// MatchEvent picks the event of the incident's meter that relates to it, or
// nil. An event relates when it lies within EventMatchHours of the onset; a
// DATA_QUALITY event relates from EventMatchHours before the onset through the
// end of the window, because a cluster onset is the first detected jump and
// may lag the event. When several relate, the highest power wins, then the
// closest to the onset, then the earliest. The winner is returned whatever its
// power so the caller can keep it as context.
func MatchEvent(inc Incident, events []domain.Event, cfg Config) *EventMatch {
	tol := time.Duration(cfg.EventMatchHours) * hour
	var best *EventMatch
	for _, e := range events {
		if e.MeterID != inc.MeterID {
			continue
		}
		e.Type = domain.NormalizeEventType(string(e.Type))
		dist := e.TS.Sub(inc.Start)
		if dist < 0 {
			dist = -dist
		}
		related := dist <= tol
		if e.Type == domain.EventDataQuality {
			related = !e.TS.Before(inc.Start.Add(-tol)) && !e.TS.After(inc.End)
		}
		if !related {
			continue
		}
		power := explanatoryPower(e.Type, inc)
		cand := EventMatch{Event: e, Power: power, Explains: power >= cfg.ExplainPowerMin && power > 0, Distance: dist}
		if best == nil || better(cand, *best) {
			c := cand
			best = &c
		}
	}
	return best
}

func better(a, b EventMatch) bool {
	if a.Power != b.Power {
		return a.Power > b.Power
	}
	if a.Distance != b.Distance {
		return a.Distance < b.Distance
	}
	if !a.Event.TS.Equal(b.Event.TS) {
		return a.Event.TS.Before(b.Event.TS)
	}
	return a.Event.ID < b.Event.ID
}
