package domain

import "time"

type AnalysisStatus string

const (
	AnalysisPending AnalysisStatus = "pending"
	AnalysisRunning AnalysisStatus = "running"
	AnalysisDone    AnalysisStatus = "done"
	AnalysisFailed  AnalysisStatus = "failed"
)

type Stage string

const (
	StageReadings       Stage = "readings"
	StageBaseline       Stage = "baseline"
	StageDetection      Stage = "detection"
	StageCorrelation    Stage = "correlation"
	StageEvents         Stage = "events"
	StageExplanation    Stage = "explanation"
	StageRecommendation Stage = "recommendation"
)

// Summary aggregates the anomalies of one analysis.
type Summary struct {
	Total             int
	PriorityAttention int
	MetersAnalyzed    int
	ByType            map[AnomalyType]int
	BySeverity        map[Severity]int
	AvgConfidence     float64
	Text              string
}

// Analysis is one run of the anomaly pipeline.
type Analysis struct {
	ID         string
	Status     AnalysisStatus
	Stage      Stage
	StageIndex int
	Progress   int
	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	Error      string
	Summary    *Summary
}
