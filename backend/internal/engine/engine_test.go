package engine

import (
	"reflect"
	"testing"

	"energy/internal/domain"
)

// forMeter returns a copy of series that belongs to meter id.
func forMeter(id string, series []domain.Reading) []domain.Reading {
	out := append([]domain.Reading(nil), series...)
	for i := range out {
		out[i].MeterID = id
	}
	return out
}

// shiftFrom scales the load by f during `hours` hours starting at day d, hour 0.
func shiftFrom(d, hours int, f float64) []domain.Reading {
	return synth(16, flatProfile(20), func(day, h int, r *domain.Reading) {
		if i := (day-d)*24 + h; i >= 0 && i < hours {
			scaleLoad(r, f)
		}
	})
}

func syntheticInput() Input {
	return Input{
		Series: map[string][]domain.Reading{
			"H-1": forMeter("H-1", synth(16, flatProfile(20), nil)),
			"R-1": forMeter("R-1", shiftFrom(8, 192, 2)),
			"E-1": forMeter("E-1", shiftFrom(8, 192, 2)),
			"S-1": forMeter("S-1", synth(3, flatProfile(20), nil)), // too short for a baseline
		},
		Events: []domain.Event{
			{ID: 1, MeterID: "E-1", TS: hrs(192), Type: domain.EventOperationalChange, Description: "new line"},
		},
	}
}

func TestAnalyzeFindsRanksAndRecommends(t *testing.T) {
	res := Analyze(syntheticInput(), DefaultConfig(), nil)

	if res.MetersAnalyzed != 3 {
		t.Fatalf("MetersAnalyzed = %d, want 3 (the short series is skipped)", res.MetersAnalyzed)
	}
	if len(res.Anomalies) != 2 {
		t.Fatalf("anomalies = %d, want 2: %+v", len(res.Anomalies), res.Anomalies)
	}
	first, second := res.Anomalies[0], res.Anomalies[1]
	if first.MeterID != "R-1" || first.Type != domain.TypeReal || first.Severity != domain.SeverityHigh || first.Rank != 1 {
		t.Fatalf("first = %s %s %s rank %d", first.MeterID, first.Type, first.Severity, first.Rank)
	}
	if second.MeterID != "E-1" || second.Type != domain.TypeExplainable || second.Severity != domain.SeverityMedium || second.Rank != 2 {
		t.Fatalf("second = %s %s %s rank %d", second.MeterID, second.Type, second.Severity, second.Rank)
	}
	if first.Action != domain.ActionInspectLoadIncrease || second.Action != domain.ActionUpdateBaseline {
		t.Fatalf("actions = %s, %s", first.Action, second.Action)
	}
	if second.RelatedEvent == nil || second.RelatedEvent.ID != 1 {
		t.Fatalf("related event = %+v", second.RelatedEvent)
	}
	for _, a := range res.Anomalies {
		if a.Reason != "" || a.RecommendedAction != "" {
			t.Fatalf("the engine leaves the texts to the Explainer: %+v", a)
		}
		if len(a.Evidence) == 0 || a.Status != "open" {
			t.Fatalf("evidence/status = %d/%s", len(a.Evidence), a.Status)
		}
	}
}

func TestAnalyzeIsDeterministic(t *testing.T) {
	in := syntheticInput()
	want := Analyze(in, DefaultConfig(), nil)
	for i := 0; i < 5; i++ {
		if got := Analyze(in, DefaultConfig(), nil); !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d differs:\n%+v\n%+v", i, got, want)
		}
	}
}

func TestAnalyzeDoesNotNeedSortedInputAndDoesNotMutateIt(t *testing.T) {
	in := syntheticInput()
	want := Analyze(in, DefaultConfig(), nil)

	rev := Input{Series: map[string][]domain.Reading{}, Events: in.Events}
	for id, s := range in.Series {
		r := append([]domain.Reading(nil), s...)
		for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
			r[i], r[j] = r[j], r[i]
		}
		rev.Series[id] = r
	}
	first := rev.Series["R-1"][0].TS
	if got := Analyze(rev, DefaultConfig(), nil); !reflect.DeepEqual(got, want) {
		t.Fatalf("reversed input gives a different result")
	}
	if !rev.Series["R-1"][0].TS.Equal(first) {
		t.Fatal("Analyze reordered the caller's slice")
	}
}

func TestAnalyzeFiresEveryEngineStageOnceInOrder(t *testing.T) {
	var got []domain.Stage
	Analyze(syntheticInput(), DefaultConfig(), func(s domain.Stage) { got = append(got, s) })
	want := []domain.Stage{
		domain.StageBaseline, domain.StageDetection, domain.StageCorrelation,
		domain.StageEvents, domain.StageExplanation, domain.StageRecommendation,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stages = %v, want %v", got, want)
	}
}

func TestAnalyzeWithNothingToReport(t *testing.T) {
	res := Analyze(Input{Series: map[string][]domain.Reading{"H-1": forMeter("H-1", synth(16, flatProfile(20), nil))}}, DefaultConfig(), nil)
	if len(res.Anomalies) != 0 || res.MetersAnalyzed != 1 {
		t.Fatalf("result = %+v", res)
	}
}

func TestOverridingTheHourlyThresholdDropsSmallerDeviations(t *testing.T) {
	// Eight hours, so the day total stays under the daily threshold and only the
	// hourly gate can report them: +47% on one meter, +100% on the other.
	in := Input{Series: map[string][]domain.Reading{
		"S-1": forMeter("S-1", shiftFrom(8, 8, 1.47)),
		"B-1": forMeter("B-1", shiftFrom(8, 8, 2)),
	}}

	def := Analyze(in, DefaultConfig(), nil)
	if len(def.Anomalies) != 2 {
		t.Fatalf("default: %d anomalies, want 2", len(def.Anomalies))
	}

	cfg := DefaultConfig()
	cfg.HourlyDevPct = 0.60
	strict := Analyze(in, cfg, nil)
	if len(strict.Anomalies) != 1 || strict.Anomalies[0].MeterID != "B-1" {
		t.Fatalf("HourlyDevPct 0.60: %+v", strict.Anomalies)
	}
}
