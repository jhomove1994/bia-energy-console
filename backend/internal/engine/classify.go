package engine

import (
	"energy/internal/domain"
)

// Recovered reports whether the incident was transient: at least
// RecoveryMinHours consecutive readings right after its end are back inside
// the hourly gate. Readings must be exactly one hour apart, so a missing
// reading ends the run. A meter without a usable baseline is never recovered.
// series must be sorted by timestamp ascending.
func Recovered(inc Incident, series []domain.Reading, b Baseline, cfg Config) bool {
	if b.InsufficientHistory || cfg.RecoveryMinHours <= 0 {
		return false
	}
	next := inc.End.Add(hour)
	run := 0
	for _, r := range series {
		if r.TS.Before(next) {
			continue
		}
		if !r.TS.Equal(next) {
			break
		}
		if _, flagged := hourFlagged(r, b, cfg); flagged {
			break
		}
		run++
		if run >= cfg.RecoveryMinHours {
			return true
		}
		next = next.Add(hour)
	}
	return false
}

// Classified is an incident with its type, kind and related event.
type Classified struct {
	Incident
	Type      domain.AnomalyType
	Kind      domain.AnomalyKind
	Recovered bool
	// Event is the event related to the incident whatever its power, nil when
	// none matched.
	Event *EventMatch
}

// Classify applies the ordered rules; the first match wins:
//
//  1. data-quality evidence in the incident: DATA_QUALITY;
//  2. a consumption incident explained by its event: FALSE_POSITIVE when it
//     recovered, EXPLAINABLE_ANOMALY when it did not;
//  3. anything else: REAL_ANOMALY.
//
// match is the result of MatchEvent; a DATA_QUALITY event never changes the
// type.
func Classify(inc Incident, match *EventMatch, recovered bool) Classified {
	out := Classified{Incident: inc, Recovered: recovered, Event: match}
	switch {
	case inc.IsDataQuality():
		out.Type = domain.TypeDataQuality
	case inc.IsConsumption() && match != nil && match.Explains && recovered:
		out.Type = domain.TypeFalsePos
	case inc.IsConsumption() && match != nil && match.Explains:
		out.Type = domain.TypeExplainable
	default:
		out.Type = domain.TypeReal
	}
	out.Kind = kindOf(inc, recovered)
	return out
}

// kindOf names the shape of the incident.
func kindOf(inc Incident, recovered bool) domain.AnomalyKind {
	up := inc.Sign >= 0
	switch {
	case inc.IsDataQuality():
		return domain.KindDataInconsistency
	case inc.PatternOnly():
		return domain.KindPatternChange
	case !inc.Persistent && up:
		return domain.KindConsumptionSpike
	case !inc.Persistent:
		return domain.KindConsumptionDip
	case recovered && up:
		return domain.KindSustainedIncrease
	case recovered:
		return domain.KindSustainedDecrease
	case up:
		return domain.KindLevelShiftUp
	}
	return domain.KindLevelShiftDown
}

// ClassifyIncident matches the events of the incident's meter, measures
// recovery on that meter's series and classifies the result. series must
// belong to inc.MeterID and be sorted; b is that meter's baseline.
func ClassifyIncident(inc Incident, events []domain.Event, series []domain.Reading, b Baseline, cfg Config) Classified {
	return Classify(inc, MatchEvent(inc, events, cfg), Recovered(inc, series, b, cfg))
}
