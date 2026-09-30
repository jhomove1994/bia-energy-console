package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"energy/internal/domain"
	"energy/internal/engine"
)

// pipelineStages are the observable stages of a run, in order. Stage k starts
// at progress floor((k-1)*100/len); done is always 100.
var pipelineStages = []domain.Stage{
	domain.StageReadings, domain.StageBaseline, domain.StageDetection, domain.StageCorrelation,
	domain.StageEvents, domain.StageExplanation, domain.StageRecommendation,
}

// Analyzer runs the engine. hook is called as each engine stage starts.
type Analyzer func(in engine.Input, cfg engine.Config, hook func(domain.Stage)) (engine.Result, error)

func defaultAnalyzer(in engine.Input, cfg engine.Config, hook func(domain.Stage)) (engine.Result, error) {
	return engine.Analyze(in, cfg, hook), nil
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

// AnalysisOption customizes an AnalysisService.
type AnalysisOption func(*AnalysisService)

// WithStageDelay sets the artificial wait at the start of every stage.
func WithStageDelay(d time.Duration) AnalysisOption { return func(s *AnalysisService) { s.delay = d } }

// WithClock replaces the clock used for timestamps.
func WithClock(c Clock) AnalysisOption { return func(s *AnalysisService) { s.clock = c } }

// WithAnalyzer replaces the engine runner (tests inject failures and pauses).
func WithAnalyzer(a Analyzer) AnalysisOption { return func(s *AnalysisService) { s.analyze = a } }

// WithLogger sets the logger used for failures that cannot reach the caller.
func WithLogger(l *slog.Logger) AnalysisOption { return func(s *AnalysisService) { s.log = l } }

// AnalysisService runs the anomaly pipeline as a single-flight background job.
type AnalysisService struct {
	analyses  AnalysisRepository
	readings  ReadingRepository
	events    EventRepository
	explainer Explainer
	ids       IDGenerator
	cfg       engine.Config

	delay   time.Duration
	clock   Clock
	analyze Analyzer
	log     *slog.Logger

	mu       sync.Mutex
	activeID string
}

// NewAnalysisService wires the service to its ports.
func NewAnalysisService(a AnalysisRepository, r ReadingRepository, e EventRepository, x Explainer, ids IDGenerator, cfg engine.Config, opts ...AnalysisOption) *AnalysisService {
	s := &AnalysisService{
		analyses: a, readings: r, events: e, explainer: x, ids: ids, cfg: cfg,
		clock: systemClock{}, analyze: defaultAnalyzer, log: slog.Default(),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Start creates a pending analysis and runs it in the background. While one is
// pending or running it returns that analysis' id instead of starting another.
func (s *AnalysisService) Start(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeID != "" {
		return s.activeID, nil
	}
	id := s.ids.NewID()
	a := domain.Analysis{ID: id, Status: domain.AnalysisPending, CreatedAt: s.clock.Now()}
	if err := s.analyses.Create(ctx, a); err != nil {
		return "", fmt.Errorf("create analysis: %w", err)
	}
	s.activeID = id
	go s.run(context.WithoutCancel(ctx), id)
	return id, nil
}

// Get returns one analysis; ErrNotFound when unknown.
func (s *AnalysisService) Get(ctx context.Context, id string) (domain.Analysis, error) {
	return s.analyses.Get(ctx, id)
}

// Latest returns the most recent analysis in any state, or nil.
func (s *AnalysisService) Latest(ctx context.Context) (*domain.Analysis, error) {
	return s.analyses.Latest(ctx)
}

// run executes one analysis. The terminal write and the release of the
// single-flight slot happen together under the lock, so a client that sees
// done can start the next run immediately.
func (s *AnalysisService) run(ctx context.Context, id string) {
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("analysis panicked", "id", id, "panic", fmt.Sprint(r))
			s.finish(func() error { return s.analyses.Fail(ctx, id, s.clock.Now(), "internal error") }, id)
		}
	}()

	sum, anomalies, err := s.execute(ctx, id)
	if err != nil {
		s.finish(func() error { return s.analyses.Fail(ctx, id, s.clock.Now(), err.Error()) }, id)
		return
	}
	s.finish(func() error { return s.analyses.Complete(ctx, id, s.clock.Now(), sum, anomalies) }, id)
}

func (s *AnalysisService) finish(write func() error, id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := write(); err != nil {
		s.log.Error("persist analysis outcome", "id", id, "error", err)
	}
	s.activeID = ""
}

func (s *AnalysisService) execute(ctx context.Context, id string) (domain.Summary, []domain.Anomaly, error) {
	enter := func(stage domain.Stage) error {
		k := stageIndex(stage)
		if err := s.analyses.UpdateProgress(ctx, id, domain.AnalysisRunning, stage, k, (k-1)*100/len(pipelineStages)); err != nil {
			return err
		}
		if s.delay > 0 {
			select {
			case <-time.After(s.delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}

	if err := enter(domain.StageReadings); err != nil {
		return domain.Summary{}, nil, err
	}
	series, err := s.readings.AllByMeter(ctx)
	if err != nil {
		return domain.Summary{}, nil, fmt.Errorf("load readings: %w", err)
	}
	events, err := s.events.All(ctx)
	if err != nil {
		return domain.Summary{}, nil, fmt.Errorf("load events: %w", err)
	}

	var hookErr error
	hook := func(stage domain.Stage) {
		if hookErr == nil {
			hookErr = enter(stage)
		}
	}
	res, err := s.analyze(engine.Input{Series: series, Events: events}, s.cfg, hook)
	if err != nil {
		return domain.Summary{}, nil, err
	}
	if hookErr != nil {
		return domain.Summary{}, nil, hookErr
	}

	anomalies := make([]domain.Anomaly, len(res.Anomalies))
	copy(anomalies, res.Anomalies)
	for i := range anomalies {
		a := &anomalies[i]
		a.ID, a.AnalysisID, a.Status = s.ids.NewID(), id, "open"
		a.Reason, a.RecommendedAction = s.explainer.Reason(*a), s.explainer.Action(*a)
	}
	return Summarize(anomalies, res.MetersAnalyzed), anomalies, nil
}

func stageIndex(stage domain.Stage) int {
	for i, st := range pipelineStages {
		if st == stage {
			return i + 1
		}
	}
	panic(errors.New("unknown analysis stage " + string(stage)))
}
