package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"energy/internal/domain"
	"energy/internal/engine"
)

// progressStep is one UpdateProgress call seen by the fake repository.
type progressStep struct {
	Status   domain.AnalysisStatus
	Stage    domain.Stage
	Index    int
	Progress int
}

// fakeAnalyses is an in-memory AnalysisRepository. Anomalies become visible
// only through Complete, like the SQL adapter.
type fakeAnalyses struct {
	mu        sync.Mutex
	rows      map[string]domain.Analysis
	order     []string
	steps     map[string][]progressStep
	anomalies map[string][]domain.Anomaly
	created   int
	completed int
}

func newFakeAnalyses() *fakeAnalyses {
	return &fakeAnalyses{rows: map[string]domain.Analysis{}, steps: map[string][]progressStep{}, anomalies: map[string][]domain.Anomaly{}}
}

func (f *fakeAnalyses) Create(_ context.Context, a domain.Analysis) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[a.ID] = a
	f.order = append(f.order, a.ID)
	f.created++
	return nil
}

func (f *fakeAnalyses) UpdateProgress(_ context.Context, id string, st domain.AnalysisStatus, stage domain.Stage, idx, progress int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := f.rows[id]
	a.Status, a.Stage, a.StageIndex, a.Progress = st, stage, idx, progress
	f.rows[id] = a
	f.steps[id] = append(f.steps[id], progressStep{st, stage, idx, progress})
	return nil
}

func (f *fakeAnalyses) Complete(_ context.Context, id string, fin time.Time, s domain.Summary, as []domain.Anomaly) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := f.rows[id]
	a.Status, a.Progress, a.FinishedAt, a.Summary = domain.AnalysisDone, 100, &fin, &s
	f.rows[id] = a
	f.anomalies[id] = as
	f.completed++
	return nil
}

func (f *fakeAnalyses) Fail(_ context.Context, id string, fin time.Time, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a := f.rows[id]
	a.Status, a.Error, a.FinishedAt = domain.AnalysisFailed, msg, &fin
	f.rows[id] = a
	return nil
}

func (f *fakeAnalyses) Get(_ context.Context, id string) (domain.Analysis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.rows[id]
	if !ok {
		return domain.Analysis{}, fmt.Errorf("analysis %s %w", id, ErrNotFound)
	}
	return a, nil
}

func (f *fakeAnalyses) Latest(context.Context) (*domain.Analysis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.order) == 0 {
		return nil, nil
	}
	a := f.rows[f.order[len(f.order)-1]]
	return &a, nil
}

func (f *fakeAnalyses) LatestDone(context.Context) (*domain.Analysis, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := len(f.order) - 1; i >= 0; i-- {
		if a := f.rows[f.order[i]]; a.Status == domain.AnalysisDone {
			return &a, nil
		}
	}
	return nil, nil
}

func (f *fakeAnalyses) FailInterrupted(context.Context, time.Time) (int, error) { return 0, nil }

func (f *fakeAnalyses) visible(id string) []domain.Anomaly {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.anomalies[id]
}

func (f *fakeAnalyses) stepsOf(id string) []progressStep {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]progressStep(nil), f.steps[id]...)
}

type seqIDs struct {
	mu sync.Mutex
	n  int
}

func (s *seqIDs) NewID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.n++
	return fmt.Sprintf("id-%d", s.n)
}

type stubExplainer struct{}

func (stubExplainer) Reason(a domain.Anomaly) string { return "reason for " + a.MeterID }
func (stubExplainer) Action(a domain.Anomaly) string { return "do " + string(a.Action) }

type failingReadings struct{ *fakeStore }

func (failingReadings) AllByMeter(context.Context) (map[string][]domain.Reading, error) {
	return nil, errors.New("disk on fire")
}

func waitFinal(t *testing.T, svc *AnalysisService, id string) domain.Analysis {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a, err := svc.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if a.Status == domain.AnalysisDone || a.Status == domain.AnalysisFailed {
			return a
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("analysis %s did not finish in time", id)
	return domain.Analysis{}
}

// twoMeterStore has 14 flat days for M-A and a tripled last day for M-B.
func twoMeterStore() *fakeStore {
	return &fakeStore{series: map[string][]domain.Reading{
		"M-A": mkSeries("M-A", 14, flat(10)),
		"M-B": mkSeries("M-B", 14, func(d, h int) float64 {
			if d == 13 {
				return 30
			}
			return 10
		}),
	}}
}

func newAnalysisService(repo *fakeAnalyses, store *fakeStore, opts ...AnalysisOption) *AnalysisService {
	return NewAnalysisService(repo, store, fakeEvents{store}, stubExplainer{}, &seqIDs{}, engine.DefaultConfig(), opts...)
}

// blockingAnalyzer waits for release, then returns two fixed anomalies.
func blockingAnalyzer(release <-chan struct{}) Analyzer {
	return func(_ engine.Input, _ engine.Config, _ func(domain.Stage)) (engine.Result, error) {
		<-release
		return engine.Result{}, nil
	}
}

func TestStartIsSingleFlight(t *testing.T) {
	repo := newFakeAnalyses()
	release := make(chan struct{})
	svc := newAnalysisService(repo, twoMeterStore(), WithAnalyzer(blockingAnalyzer(release)))

	const callers = 8
	ids := make([]string, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, err := svc.Start(context.Background())
			if err != nil {
				t.Error(err)
			}
			ids[i] = id
		}(i)
	}
	wg.Wait()
	for _, id := range ids {
		if id == "" || id != ids[0] {
			t.Fatalf("all callers must share one id, got %v", ids)
		}
	}
	if repo.created != 1 {
		t.Fatalf("rows created = %d, want 1", repo.created)
	}
	if a, err := svc.Get(context.Background(), ids[0]); err != nil || (a.Status != domain.AnalysisPending && a.Status != domain.AnalysisRunning) {
		t.Fatalf("active analysis = %+v, %v", a, err)
	}

	close(release)
	if a := waitFinal(t, svc, ids[0]); a.Status != domain.AnalysisDone {
		t.Fatalf("final = %+v", a)
	}
	next, err := svc.Start(context.Background())
	if err != nil || next == ids[0] {
		t.Fatalf("Start after done = %q, %v; want a new id", next, err)
	}
	waitFinal(t, svc, next)
	if repo.created != 2 {
		t.Fatalf("rows created = %d, want 2", repo.created)
	}
}

func TestStageOrderAndProgress(t *testing.T) {
	repo := newFakeAnalyses()
	svc := newAnalysisService(repo, twoMeterStore())
	id, err := svc.Start(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	final := waitFinal(t, svc, id)

	wantStages := []domain.Stage{domain.StageReadings, domain.StageBaseline, domain.StageDetection, domain.StageCorrelation,
		domain.StageEvents, domain.StageExplanation, domain.StageRecommendation}
	wantProgress := []int{0, 14, 28, 42, 57, 71, 85}
	steps := repo.stepsOf(id)
	if len(steps) != 7 {
		t.Fatalf("steps = %+v, want 7", steps)
	}
	last := -1
	for i, s := range steps {
		if s.Stage != wantStages[i] || s.Progress != wantProgress[i] || s.Index != i+1 || s.Status != domain.AnalysisRunning {
			t.Fatalf("step %d = %+v, want %s %d idx %d running", i, s, wantStages[i], wantProgress[i], i+1)
		}
		if s.Progress < last {
			t.Fatalf("progress decreased at step %d: %+v", i, steps)
		}
		last = s.Progress
	}
	if final.Status != domain.AnalysisDone || final.Progress != 100 || final.Error != "" || final.Summary == nil || final.FinishedAt == nil {
		t.Fatalf("final = %+v", final)
	}
}

func TestDoneOnlyAtFullProgress(t *testing.T) {
	repo := newFakeAnalyses()
	release := make(chan struct{})
	svc := newAnalysisService(repo, twoMeterStore(), WithAnalyzer(blockingAnalyzer(release)))
	id, _ := svc.Start(context.Background())
	// The analyzer is blocked: the run cannot be done and nothing is visible.
	time.Sleep(20 * time.Millisecond)
	a, _ := svc.Get(context.Background(), id)
	if a.Status == domain.AnalysisDone || a.Progress >= 100 || len(repo.visible(id)) != 0 {
		t.Fatalf("running analysis must not look done: %+v", a)
	}
	close(release)
	if final := waitFinal(t, svc, id); final.Status != domain.AnalysisDone || final.Progress != 100 {
		t.Fatalf("final = %+v", final)
	}
}

func TestResultsBecomeVisibleAtDoneWithExplanationsAndSummary(t *testing.T) {
	repo := newFakeAnalyses()
	fixed := func(_ engine.Input, _ engine.Config, hook func(domain.Stage)) (engine.Result, error) {
		hook(domain.StageBaseline)
		return engine.Result{MetersAnalyzed: 7, Anomalies: []domain.Anomaly{
			{MeterID: "M-B", Type: domain.TypeReal, Severity: domain.SeverityHigh, Confidence: 0.9, Rank: 1, Action: domain.ActionInspectLoadIncrease},
			{MeterID: "M-A", Type: domain.TypeFalsePos, Severity: domain.SeverityLow, Confidence: 0.6, Rank: 2, Action: domain.ActionAcknowledgeExpected},
		}}, nil
	}
	svc := newAnalysisService(repo, twoMeterStore(), WithAnalyzer(fixed))
	id, _ := svc.Start(context.Background())
	final := waitFinal(t, svc, id)

	as := repo.visible(id)
	if len(as) != 2 || as[0].MeterID != "M-B" || as[1].MeterID != "M-A" {
		t.Fatalf("anomalies = %+v", as)
	}
	if as[0].ID == "" || as[0].ID == as[1].ID || as[0].AnalysisID != id || as[1].AnalysisID != id {
		t.Fatalf("ids not assigned: %+v", as)
	}
	if as[0].Reason != "reason for M-B" || as[0].RecommendedAction != "do INSPECT_LOAD_INCREASE" || as[1].RecommendedAction != "do ACKNOWLEDGE_EXPECTED" {
		t.Fatalf("explainer not applied: %+v", as)
	}
	if as[0].Status != "open" {
		t.Fatalf("status = %q, want open", as[0].Status)
	}
	s := final.Summary
	if s == nil || s.Total != 2 || s.PriorityAttention != 1 || s.MetersAnalyzed != 7 || s.Text != "2 anomalías detectadas · 1 requiere atención prioritaria" {
		t.Fatalf("summary = %+v", s)
	}
}

func TestRealEngineFindsTheShiftedMeter(t *testing.T) {
	repo := newFakeAnalyses()
	svc := newAnalysisService(repo, twoMeterStore())
	id, _ := svc.Start(context.Background())
	final := waitFinal(t, svc, id)
	if final.Status != domain.AnalysisDone {
		t.Fatalf("final = %+v", final)
	}
	as := repo.visible(id)
	if len(as) != 1 || as[0].MeterID != "M-B" || as[0].Reason == "" || as[0].RecommendedAction == "" {
		t.Fatalf("anomalies = %+v, want one explained anomaly on M-B", as)
	}
}

func TestFailureLeavesNoAnomaliesAndDoesNotBlockNextRun(t *testing.T) {
	failures := map[string]Analyzer{
		"engine error": func(engine.Input, engine.Config, func(domain.Stage)) (engine.Result, error) {
			return engine.Result{}, errors.New("engine exploded")
		},
		"panic": func(engine.Input, engine.Config, func(domain.Stage)) (engine.Result, error) { panic("kaboom") },
	}
	for name, failing := range failures {
		t.Run(name, func(t *testing.T) {
			repo := newFakeAnalyses()
			var mu sync.Mutex
			calls := 0
			analyzer := func(in engine.Input, cfg engine.Config, hook func(domain.Stage)) (engine.Result, error) {
				mu.Lock()
				calls++
				n := calls
				mu.Unlock()
				if n == 1 {
					return failing(in, cfg, hook)
				}
				return engine.Result{}, nil
			}
			svc := newAnalysisService(repo, twoMeterStore(), WithAnalyzer(analyzer))

			id, _ := svc.Start(context.Background())
			a := waitFinal(t, svc, id)
			if a.Status != domain.AnalysisFailed || a.Error == "" || repo.completed != 0 || len(repo.visible(id)) != 0 || a.Summary != nil {
				t.Fatalf("failed run = %+v (completed=%d)", a, repo.completed)
			}
			if name == "engine error" && a.Error != "engine exploded" {
				t.Fatalf("error = %q", a.Error)
			}
			retry, err := svc.Start(context.Background())
			if err != nil || retry == id {
				t.Fatalf("Start after failure = %q, %v; want a new id", retry, err)
			}
			if b := waitFinal(t, svc, retry); b.Status != domain.AnalysisDone {
				t.Fatalf("retry = %+v", b)
			}
		})
	}
}

func TestReadingsErrorFailsTheRun(t *testing.T) {
	repo := newFakeAnalyses()
	store := twoMeterStore()
	svc := NewAnalysisService(repo, failingReadings{store}, fakeEvents{store}, stubExplainer{}, &seqIDs{}, engine.DefaultConfig())
	id, _ := svc.Start(context.Background())
	a := waitFinal(t, svc, id)
	if a.Status != domain.AnalysisFailed || a.Error == "" || len(repo.visible(id)) != 0 {
		t.Fatalf("run = %+v", a)
	}
}

func TestStageDelayIsObservable(t *testing.T) {
	repo := newFakeAnalyses()
	svc := newAnalysisService(repo, twoMeterStore(), WithStageDelay(25*time.Millisecond))
	id, _ := svc.Start(context.Background())

	stages := map[domain.Stage]bool{}
	last := -1
	for {
		a, err := svc.Get(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if a.Progress < last {
			t.Fatalf("progress decreased: %d after %d", a.Progress, last)
		}
		last = a.Progress
		if a.Stage != "" {
			stages[a.Stage] = true
		}
		if a.Status == domain.AnalysisDone || a.Status == domain.AnalysisFailed {
			if a.Status != domain.AnalysisDone || a.Progress != 100 {
				t.Fatalf("final = %+v", a)
			}
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(stages) < 2 {
		t.Fatalf("observed stages %v, want at least two", stages)
	}
}

func TestLatestReturnsMostRecent(t *testing.T) {
	repo := newFakeAnalyses()
	svc := newAnalysisService(repo, twoMeterStore())
	if a, err := svc.Latest(context.Background()); err != nil || a != nil {
		t.Fatalf("Latest before any run = %v, %v", a, err)
	}
	id, _ := svc.Start(context.Background())
	waitFinal(t, svc, id)
	if a, err := svc.Latest(context.Background()); err != nil || a == nil || a.ID != id {
		t.Fatalf("Latest = %+v, %v; want %s", a, err, id)
	}
	if _, err := svc.Get(context.Background(), "nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(nope) = %v, want ErrNotFound", err)
	}
}
