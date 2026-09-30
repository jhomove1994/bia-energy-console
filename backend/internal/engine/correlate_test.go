package engine

import (
	"testing"
	"time"

	"energy/internal/domain"
)

// hrs is the instant `h` hours after testStart.
func hrs(h int) time.Time { return testStart.Add(time.Duration(h) * time.Hour) }

// mkSig builds a signal of one meter covering [from, to] hours after testStart.
func mkSig(meter string, kind SignalKind, role Role, from, to int) Signal {
	return Signal{
		MeterID: meter,
		Kind:    kind,
		Role:    role,
		Start:   hrs(from),
		End:     hrs(to),
		Hours:   to - from + 1,
	}
}

func seed(kind SignalKind, from, to int) Signal { return mkSig("T-1", kind, RoleSeed, from, to) }
func secondary(kind SignalKind, from, to int) Signal {
	return mkSig("T-1", kind, RoleSecondary, from, to)
}

func kindsOf(sigs []Signal) []SignalKind {
	out := make([]SignalKind, len(sigs))
	for i, s := range sigs {
		out[i] = s.Kind
	}
	return out
}

func TestSeveralDetectorsOnOneWindowGiveOneIncident(t *testing.T) {
	// A persistent shift on day 8 (hours 200..223) with every kind of
	// secondary evidence on the same window.
	shift := seed(KindPersistentShift, 200, 223)
	shift.Sign = 1
	sigs := []Signal{
		shift,
		secondary(KindDailyVariation, 192, 215),
		secondary(KindJump, 200, 200),
		secondary(KindRatioStable, 200, 223),
		secondary(KindPatternBreak, 192, 215),
	}
	got := Correlate(sigs, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1: %+v", len(got), got)
	}
	inc := got[0]
	if !inc.Start.Equal(hrs(200)) || !inc.End.Equal(hrs(223)) {
		t.Fatalf("window = %v..%v, want %v..%v", inc.Start, inc.End, hrs(200), hrs(223))
	}
	if len(inc.Seeds) != 1 || len(inc.Attached) != 4 {
		t.Fatalf("seeds=%d attached=%d, want 1 and 4", len(inc.Seeds), len(inc.Attached))
	}
	if !inc.Persistent || inc.Sign != 1 {
		t.Fatalf("persistent=%v sign=%d", inc.Persistent, inc.Sign)
	}
}

func TestSeparatedWindowsGiveSeparateIncidents(t *testing.T) {
	sigs := []Signal{
		seed(KindPersistentShift, 200, 223),
		secondary(KindDailyVariation, 192, 215),
		seed(KindPersistentShift, 400, 430),
		secondary(KindDailyVariation, 408, 431),
	}
	got := Correlate(sigs, DefaultConfig())
	if len(got) != 2 {
		t.Fatalf("incidents = %d, want 2", len(got))
	}
	if !got[0].Start.Equal(hrs(200)) || !got[1].Start.Equal(hrs(400)) {
		t.Fatalf("starts = %v, %v; want ordered by time", got[0].Start, got[1].Start)
	}
	if len(got[0].Attached) != 1 || len(got[1].Attached) != 1 {
		t.Fatalf("each incident keeps only its own secondary: %d, %d", len(got[0].Attached), len(got[1].Attached))
	}
}

func TestSecondariesAttachButNeverExtendTheWindow(t *testing.T) {
	// The daily variation spans a whole day around a 6 hour spike; the
	// pattern break ends 2 hours before the seed (inside the slack).
	sigs := []Signal{
		seed(KindSpike, 210, 215),
		secondary(KindDailyVariation, 192, 215),
		secondary(KindPatternBreak, 180, 208),
	}
	got := Correlate(sigs, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1", len(got))
	}
	if !got[0].Start.Equal(hrs(210)) || !got[0].End.Equal(hrs(215)) {
		t.Fatalf("window = %v..%v, want the seed window only", got[0].Start, got[0].End)
	}
	if len(got[0].Attached) != 2 {
		t.Fatalf("attached = %v, want both", kindsOf(got[0].Attached))
	}
}

func TestSecondaryBeyondTheSlackDoesNotAttach(t *testing.T) {
	cfg := DefaultConfig()
	// The jump is 3 hours before the seed starts, one more than
	// AttachSlackHours (2): it is an orphan single jump and is dropped.
	sigs := []Signal{seed(KindSpike, 210, 215), secondary(KindJump, 207, 207)}
	got := Correlate(sigs, cfg)
	if len(got) != 1 || len(got[0].Attached) != 0 {
		t.Fatalf("incidents=%d attached=%v, want 1 incident with no evidence", len(got), got)
	}
	// One hour closer it attaches.
	sigs = []Signal{seed(KindSpike, 210, 215), secondary(KindJump, 208, 208)}
	got = Correlate(sigs, cfg)
	if len(got) != 1 || len(got[0].Attached) != 1 {
		t.Fatalf("attached = %+v, want the jump inside the slack", got)
	}
	// The same distance after the window end.
	sigs = []Signal{seed(KindSpike, 210, 215), secondary(KindJump, 217, 217), secondary(KindJump, 218, 218)}
	got = Correlate(sigs, cfg)
	if len(got) != 1 || len(got[0].Attached) != 1 || !got[0].Attached[0].Start.Equal(hrs(217)) {
		t.Fatalf("attached = %+v, want only the jump 2 h after the end", got)
	}
}

func TestSeedsWithinAttachSlackMerge(t *testing.T) {
	cfg := DefaultConfig()
	// Seeds at most AttachSlackHours (2) apart, measured from one window end
	// to the next start, merge.
	merged := Correlate([]Signal{seed(KindSpike, 100, 105), seed(KindDQCluster, 107, 120)}, cfg)
	if len(merged) != 1 {
		t.Fatalf("incidents = %d, want 1 (2 h apart)", len(merged))
	}
	if !merged[0].Start.Equal(hrs(100)) || !merged[0].End.Equal(hrs(120)) || len(merged[0].Seeds) != 2 {
		t.Fatalf("merged = %+v, want union 100..120 with 2 seeds", merged[0])
	}
	apart := Correlate([]Signal{seed(KindSpike, 100, 105), seed(KindDQCluster, 108, 120)}, cfg)
	if len(apart) != 2 {
		t.Fatalf("incidents = %d, want 2 (3 h apart)", len(apart))
	}
	cfg.AttachSlackHours = 3
	if got := Correlate([]Signal{seed(KindSpike, 100, 105), seed(KindDQCluster, 108, 120)}, cfg); len(got) != 1 {
		t.Fatalf("with AttachSlackHours=3 incidents = %d, want 1", len(got))
	}
}

func TestOverlappingSeedsMergeAndUseEarliestOnset(t *testing.T) {
	got := Correlate([]Signal{
		seed(KindPersistentShift, 300, 340),
		seed(KindRatioUnstable, 290, 310),
	}, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1", len(got))
	}
	if !got[0].Start.Equal(hrs(290)) || !got[0].End.Equal(hrs(340)) {
		t.Fatalf("window = %v..%v, want 290..340", got[0].Start, got[0].End)
	}
}

func TestOrphanDailyVariationPromotesToConsumptionIncident(t *testing.T) {
	day := secondary(KindDailyVariation, 192, 215)
	day.Sign = -1
	got := Correlate([]Signal{day}, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1", len(got))
	}
	inc := got[0]
	if !inc.Start.Equal(hrs(192)) || !inc.End.Equal(hrs(215)) {
		t.Fatalf("window = %v..%v, want the day", inc.Start, inc.End)
	}
	if inc.Sign != -1 || !inc.Persistent || !inc.IsConsumption() || inc.IsDataQuality() {
		t.Fatalf("sign=%d persistent=%v consumption=%v dq=%v", inc.Sign, inc.Persistent, inc.IsConsumption(), inc.IsDataQuality())
	}
	if len(inc.Seeds) != 1 || inc.Seeds[0].Kind != KindDailyVariation {
		t.Fatalf("seeds = %v, want the promoted daily variation", kindsOf(inc.Seeds))
	}
}

func TestAdjacentOrphanDaysMergeAndPatternBreakAttachesToPromotedDay(t *testing.T) {
	d1 := secondary(KindDailyVariation, 192, 215)
	d1.Sign = 1
	d2 := secondary(KindDailyVariation, 216, 239)
	d2.Sign = 1
	got := Correlate([]Signal{d2, secondary(KindPatternBreak, 192, 215), d1}, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1 (adjacent days and a covered pattern break)", len(got))
	}
	if !got[0].Start.Equal(hrs(192)) || !got[0].End.Equal(hrs(239)) {
		t.Fatalf("window = %v..%v, want both days", got[0].Start, got[0].End)
	}
	if len(got[0].Seeds) != 2 || len(got[0].Attached) != 1 || got[0].Attached[0].Kind != KindPatternBreak {
		t.Fatalf("seeds=%v attached=%v", kindsOf(got[0].Seeds), kindsOf(got[0].Attached))
	}
}

func TestOrphanPatternBreakBecomesPatternChangeIncident(t *testing.T) {
	got := Correlate([]Signal{secondary(KindPatternBreak, 192, 215)}, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1", len(got))
	}
	inc := got[0]
	if !inc.PatternOnly() || inc.IsConsumption() || inc.IsDataQuality() || inc.Sign != 0 || inc.Persistent {
		t.Fatalf("pattern-only=%v consumption=%v dq=%v sign=%d persistent=%v",
			inc.PatternOnly(), inc.IsConsumption(), inc.IsDataQuality(), inc.Sign, inc.Persistent)
	}
	// Triangulation: covered by a consumption seed, the same break is only
	// evidence and the incident is not pattern-only.
	with := Correlate([]Signal{seed(KindSpike, 200, 205), secondary(KindPatternBreak, 192, 215)}, DefaultConfig())
	if len(with) != 1 || with[0].PatternOnly() {
		t.Fatalf("incidents=%d patternOnly=%v, want one consumption incident", len(with), len(with) == 1 && with[0].PatternOnly())
	}
}

func TestOrphanSingleJumpsAreDropped(t *testing.T) {
	got := Correlate([]Signal{secondary(KindJump, 100, 100), secondary(KindJump, 300, 300)}, DefaultConfig())
	if len(got) != 0 {
		t.Fatalf("incidents = %d, want 0 for orphan single jumps", len(got))
	}
}

func TestOrphanStableRatioNeedsToBeStandalone(t *testing.T) {
	short := secondary(KindRatioStable, 100, 101)
	short.Standalone = false
	if got := Correlate([]Signal{short}, DefaultConfig()); len(got) != 0 {
		t.Fatalf("incidents = %d, want 0 for a short orphan ratio run", len(got))
	}

	long := secondary(KindRatioStable, 100, 130)
	long.Standalone = true
	got := Correlate([]Signal{long}, DefaultConfig())
	if len(got) != 1 {
		t.Fatalf("incidents = %d, want 1 drift incident for a standalone ratio run", len(got))
	}
	if !got[0].IsDataQuality() || got[0].IsConsumption() || got[0].Sign != 0 {
		t.Fatalf("dq=%v consumption=%v sign=%d, want a DQ incident", got[0].IsDataQuality(), got[0].IsConsumption(), got[0].Sign)
	}
}

func TestStandaloneRatioCoveredByAConsumptionSeedIsOnlyEvidence(t *testing.T) {
	long := secondary(KindRatioStable, 200, 258)
	long.Standalone = true
	got := Correlate([]Signal{seed(KindPersistentShift, 200, 258), long}, DefaultConfig())
	if len(got) != 1 || got[0].IsDataQuality() || len(got[0].Attached) != 1 {
		t.Fatalf("incidents=%d, want one consumption incident carrying the ratio as evidence: %+v", len(got), got)
	}
}

func TestIncidentsAreSeparatePerMeterAndOrdered(t *testing.T) {
	sigs := []Signal{
		mkSig("T-2", KindPersistentShift, RoleSeed, 200, 223),
		mkSig("T-1", KindSpike, RoleSeed, 300, 305),
		mkSig("T-1", KindSpike, RoleSeed, 200, 205),
		mkSig("T-2", KindJump, RoleSecondary, 200, 200),
	}
	got := Correlate(sigs, DefaultConfig())
	if len(got) != 3 {
		t.Fatalf("incidents = %d, want 3", len(got))
	}
	want := []struct {
		meter string
		start int
	}{{"T-1", 200}, {"T-1", 300}, {"T-2", 200}}
	for i, w := range want {
		if got[i].MeterID != w.meter || !got[i].Start.Equal(hrs(w.start)) {
			t.Fatalf("incident %d = %s@%v, want %s@%v", i, got[i].MeterID, got[i].Start, w.meter, hrs(w.start))
		}
	}
	if len(got[2].Attached) != 1 {
		t.Fatalf("T-2 attached = %d, want its own jump", len(got[2].Attached))
	}
}

func TestIncidentTraitsFollowTheSeeds(t *testing.T) {
	down := seed(KindPersistentShift, 200, 211)
	down.Sign = -1
	inc := Correlate([]Signal{down}, DefaultConfig())[0]
	if inc.Sign != -1 || !inc.Persistent || !inc.IsConsumption() {
		t.Fatalf("persistent shift: sign=%d persistent=%v", inc.Sign, inc.Persistent)
	}
	spike := seed(KindSpike, 200, 203)
	spike.Sign = 1
	inc = Correlate([]Signal{spike}, DefaultConfig())[0]
	if inc.Sign != 1 || inc.Persistent {
		t.Fatalf("spike: sign=%d persistent=%v, want +1 and not persistent", inc.Sign, inc.Persistent)
	}
	for _, k := range []SignalKind{KindDQCluster, KindRatioUnstable, KindGap, KindStatus} {
		inc = Correlate([]Signal{seed(k, 200, 210)}, DefaultConfig())[0]
		if !inc.IsDataQuality() || inc.IsConsumption() || inc.Sign != 0 {
			t.Fatalf("%s: dq=%v consumption=%v sign=%d", k, inc.IsDataQuality(), inc.IsConsumption(), inc.Sign)
		}
	}
}

func TestDetectorsToIncidentsOnASyntheticMeter(t *testing.T) {
	cfg := DefaultConfig()
	// A quiet baseline week, then +50% for hours 10..23 of day 8 and the
	// whole of day 9 (persistent), and an unrelated spike on day 14.
	series := synth(16, shapedProfile(), func(d, h int, r *domain.Reading) {
		if (d == 8 && h >= 10) || d == 9 {
			scaleLoad(r, 1.5)
		}
		if d == 14 && h == 12 {
			scaleLoad(r, 2.0)
		}
	})
	got := Correlate(allSignals(series, cfg), cfg)
	if len(got) != 2 {
		t.Fatalf("incidents = %d, want 2 (shift and spike): %+v", len(got), got)
	}
	if !got[0].Persistent || got[0].Sign != 1 {
		t.Fatalf("first incident persistent=%v sign=%d, want persistent +1", got[0].Persistent, got[0].Sign)
	}
	if got[1].Persistent || got[1].Sign != 1 {
		t.Fatalf("second incident persistent=%v sign=%d, want a non-persistent +1 spike", got[1].Persistent, got[1].Sign)
	}
}

// allSignals runs every detector over one meter's series.
func allSignals(series []domain.Reading, cfg Config) []Signal {
	b := BuildBaseline(series, cfg)
	var out []Signal
	out = append(out, DetectConsumption(series, b, cfg)...)
	out = append(out, DetectDailyVariation(series, b, cfg)...)
	out = append(out, DetectJumps(series, cfg)...)
	out = append(out, DetectRatio(series, b, cfg)...)
	out = append(out, DetectPatternBreaks(series, b, cfg)...)
	out = append(out, DetectGaps(series)...)
	out = append(out, DetectStatus(series)...)
	return out
}

func TestOrphanJumpAttachesToAPromotedIncident(t *testing.T) {
	day := secondary(KindDailyVariation, 192, 215)
	day.Sign = 1
	got := Correlate([]Signal{day, secondary(KindJump, 200, 200), secondary(KindJump, 400, 400)}, DefaultConfig())
	if len(got) != 1 || len(got[0].Attached) != 1 || !got[0].Attached[0].Start.Equal(hrs(200)) {
		t.Fatalf("incidents=%+v, want one promoted day carrying only the jump inside it", got)
	}
}
