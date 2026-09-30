package engine

import (
	"testing"

	"energy/internal/domain"
)

func evAt(id int64, h int, typ string) domain.Event {
	return domain.Event{ID: id, MeterID: "T-1", TS: hrs(h), Type: domain.EventType(typ), Description: "event " + typ}
}

// incidentOf builds the incident Correlate makes of the given signals of T-1.
func incidentOf(t *testing.T, sigs ...Signal) Incident {
	t.Helper()
	got := Correlate(sigs, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1", len(got))
	}
	return got[0]
}

// consumptionIncident is a persistent shift (or a spike when persistent is
// false) of the given sign covering hours 200..211.
func consumptionIncident(t *testing.T, sgn int, persistent bool) Incident {
	t.Helper()
	kind := KindSpike
	if persistent {
		kind = KindPersistentShift
	}
	s := seed(kind, 200, 211)
	s.Sign = sgn
	return incidentOf(t, s)
}

func mustMatch(t *testing.T, inc Incident, events []domain.Event) *EventMatch {
	t.Helper()
	m := MatchEvent(inc, events, DefaultConfig())
	if m == nil {
		t.Fatalf("no event matched, want one")
	}
	return m
}

func TestOutageAtOnsetExplainsADecrease(t *testing.T) {
	m := mustMatch(t, consumptionIncident(t, -1, true), []domain.Event{evAt(1, 200, "SCHEDULED_OUTAGE")})
	if m.Event.ID != 1 || m.Power != 1.0 || !m.Explains {
		t.Fatalf("match = %+v, want event 1 with power 1 explaining", m)
	}
}

func TestOutageDoesNotExplainAnIncrease(t *testing.T) {
	m := mustMatch(t, consumptionIncident(t, 1, true), []domain.Event{evAt(1, 200, "SCHEDULED_OUTAGE")})
	if m.Power != 0 || m.Explains {
		t.Fatalf("match = %+v, want the outage related but powerless on an increase", m)
	}
}

func TestUnknownEventIsContextOnly(t *testing.T) {
	m := mustMatch(t, consumptionIncident(t, 1, true), []domain.Event{evAt(7, 200, "UNKNOWN")})
	if m.Event.ID != 7 || m.Power != 0 || m.Explains {
		t.Fatalf("match = %+v, want the UNKNOWN event related without explaining", m)
	}
}

func TestOperationalChangeExplainsPersistentShiftsOfBothSigns(t *testing.T) {
	for _, sgn := range []int{1, -1} {
		m := mustMatch(t, consumptionIncident(t, sgn, true), []domain.Event{evAt(1, 200, "OPERATIONAL_CHANGE")})
		if m.Power != 0.9 || !m.Explains {
			t.Fatalf("sign %d: match = %+v, want power 0.9 explaining", sgn, m)
		}
	}
	// A spike is not a persistent shift.
	m := mustMatch(t, consumptionIncident(t, 1, false), []domain.Event{evAt(1, 200, "OPERATIONAL_CHANGE")})
	if m.Power != 0 || m.Explains {
		t.Fatalf("spike: match = %+v, want no explanation", m)
	}
}

func TestExplainPowerMinIsConfigurable(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ExplainPowerMin = 0.95
	m := MatchEvent(consumptionIncident(t, 1, true), []domain.Event{evAt(1, 200, "OPERATIONAL_CHANGE")}, cfg)
	if m == nil || m.Power != 0.9 || m.Explains {
		t.Fatalf("match = %+v, want power 0.9 below the raised minimum", m)
	}
}

func TestEventMatchesOnlyWithinTheToleranceOfTheOnset(t *testing.T) {
	inc := consumptionIncident(t, -1, true)
	cases := []struct {
		name string
		hour int
		want bool
	}{
		{"exactly at onset", 200, true},
		{"2 h before", 198, true},
		{"2 h after", 202, true},
		{"3 h before", 197, false},
		{"3 h after", 203, false},
		{"10 h before", 190, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchEvent(inc, []domain.Event{evAt(1, tt.hour, "SCHEDULED_OUTAGE")}, DefaultConfig())
			if (got != nil) != tt.want {
				t.Fatalf("matched = %v, want %v", got != nil, tt.want)
			}
		})
	}
	cfg := DefaultConfig()
	cfg.EventMatchHours = 4
	if MatchEvent(inc, []domain.Event{evAt(1, 197, "SCHEDULED_OUTAGE")}, cfg) == nil {
		t.Fatalf("EventMatchHours=4 must match an event 3 h before the onset")
	}
}

func TestEventsOfOtherMetersAreIgnored(t *testing.T) {
	other := evAt(1, 200, "SCHEDULED_OUTAGE")
	other.MeterID = "T-2"
	if got := MatchEvent(consumptionIncident(t, -1, true), []domain.Event{other}, DefaultConfig()); got != nil {
		t.Fatalf("matched an event of another meter: %+v", got)
	}
}

func TestDataQualityEventMatchesFromTwoHoursBeforeOnsetToTheWindowEnd(t *testing.T) {
	inc := incidentOf(t, seed(KindDQCluster, 200, 247))
	cases := []struct {
		name string
		hour int
		want bool
	}{
		{"2 h before onset", 198, true},
		{"3 h before onset", 197, false},
		{"mid window, later than the onset tolerance", 230, true},
		{"at the window end", 247, true},
		{"after the window end", 248, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchEvent(inc, []domain.Event{evAt(1, tt.hour, "DATA_QUALITY")}, DefaultConfig())
			if (got != nil) != tt.want {
				t.Fatalf("matched = %v, want %v", got != nil, tt.want)
			}
			if got != nil && (got.Power != 0 || got.Explains) {
				t.Fatalf("DATA_QUALITY event = %+v, must never explain", got)
			}
		})
	}
	// A non-DATA_QUALITY event keeps the strict onset rule on the same window.
	if MatchEvent(inc, []domain.Event{evAt(1, 230, "UNKNOWN")}, DefaultConfig()) != nil {
		t.Fatalf("UNKNOWN event mid window must not match")
	}
}

func TestEventTypesAreNormalizedBeforeMatching(t *testing.T) {
	cases := []struct {
		raw  string
		want domain.EventType
	}{
		{"maintenance", domain.EventScheduledOutage},
		{"  planned_outage ", domain.EventScheduledOutage},
		{"Scheduled_Maintenance", domain.EventScheduledOutage},
		{"operational_change", domain.EventOperationalChange},
		{"FIRMWARE_UPDATE", domain.EventUnknown},
	}
	inc := consumptionIncident(t, -1, true)
	for _, tt := range cases {
		t.Run(tt.raw, func(t *testing.T) {
			m := mustMatch(t, inc, []domain.Event{evAt(1, 200, tt.raw)})
			if m.Event.Type != tt.want {
				t.Fatalf("type = %q, want %q", m.Event.Type, tt.want)
			}
		})
	}
	if m := mustMatch(t, inc, []domain.Event{evAt(1, 200, "maintenance")}); !m.Explains {
		t.Fatalf("maintenance must carry outage semantics on a decrease")
	}
}

func TestBestEventWinsByPowerThenCloseness(t *testing.T) {
	inc := consumptionIncident(t, -1, true)

	// Power beats closeness: an outage 2 h away over an exact UNKNOWN.
	m := mustMatch(t, inc, []domain.Event{evAt(1, 200, "UNKNOWN"), evAt(2, 202, "SCHEDULED_OUTAGE")})
	if m.Event.ID != 2 {
		t.Fatalf("winner = %d, want the outage (power 1.0) 2 h away", m.Event.ID)
	}
	// Outage (1.0) beats operational change (0.9) whatever the order.
	m = mustMatch(t, inc, []domain.Event{evAt(3, 200, "OPERATIONAL_CHANGE"), evAt(4, 201, "SCHEDULED_OUTAGE")})
	if m.Event.ID != 4 {
		t.Fatalf("winner = %d, want the outage", m.Event.ID)
	}
	// Equal power: the closest to the onset wins.
	m = mustMatch(t, inc, []domain.Event{evAt(5, 202, "SCHEDULED_OUTAGE"), evAt(6, 199, "SCHEDULED_OUTAGE")})
	if m.Event.ID != 6 {
		t.Fatalf("winner = %d, want the outage 1 h from the onset", m.Event.ID)
	}
	// Equal power and equal distance: deterministic, earliest event first.
	m = mustMatch(t, inc, []domain.Event{evAt(8, 201, "SCHEDULED_OUTAGE"), evAt(9, 199, "SCHEDULED_OUTAGE")})
	if m.Event.ID != 9 {
		t.Fatalf("winner = %d, want the earlier event on a full tie", m.Event.ID)
	}
	// Direction-aware: on an increase both are powerless, so closeness decides.
	up := consumptionIncident(t, 1, true)
	m = mustMatch(t, up, []domain.Event{evAt(10, 202, "SCHEDULED_OUTAGE"), evAt(11, 200, "UNKNOWN")})
	if m.Event.ID != 11 {
		t.Fatalf("winner = %d, want the closest event when none has power", m.Event.ID)
	}
}
