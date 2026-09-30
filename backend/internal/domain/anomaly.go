package domain

import "time"

type AnomalyType string

const (
	TypeReal        AnomalyType = "REAL_ANOMALY"
	TypeExplainable AnomalyType = "EXPLAINABLE_ANOMALY"
	TypeFalsePos    AnomalyType = "FALSE_POSITIVE"
	TypeDataQuality AnomalyType = "DATA_QUALITY"
)

type Severity string

const (
	SeverityHigh   Severity = "HIGH"
	SeverityMedium Severity = "MEDIUM"
	SeverityLow    Severity = "LOW"
)

type AnomalyKind string

const (
	KindConsumptionSpike  AnomalyKind = "CONSUMPTION_SPIKE"
	KindConsumptionDip    AnomalyKind = "CONSUMPTION_DIP"
	KindSustainedIncrease AnomalyKind = "SUSTAINED_INCREASE"
	KindSustainedDecrease AnomalyKind = "SUSTAINED_DECREASE"
	KindLevelShiftUp      AnomalyKind = "LEVEL_SHIFT_UP"
	KindLevelShiftDown    AnomalyKind = "LEVEL_SHIFT_DOWN"
	KindDataInconsistency AnomalyKind = "DATA_INCONSISTENCY"
	KindPatternChange     AnomalyKind = "PATTERN_CHANGE"
)

type ActionCode string

const (
	ActionInspectLoadIncrease ActionCode = "INSPECT_LOAD_INCREASE"
	ActionCheckSupplyLoss     ActionCode = "CHECK_SUPPLY_LOSS"
	ActionReviewTransient     ActionCode = "REVIEW_TRANSIENT"
	ActionOpenDQTicket        ActionCode = "OPEN_DQ_TICKET"
	ActionUpdateBaseline      ActionCode = "UPDATE_BASELINE"
	ActionAcknowledgeExpected ActionCode = "ACKNOWLEDGE_EXPECTED"
	ActionReviewSchedule      ActionCode = "REVIEW_SCHEDULE"
)

// Anomaly is one finding per meter per incident window.
type Anomaly struct {
	ID, AnalysisID, MeterID   string
	DetectedAt                time.Time // equals WindowStart
	WindowStart, WindowEnd    time.Time
	Kind                      AnomalyKind
	Type                      AnomalyType
	Severity                  Severity
	SeverityScore             float64 // 0-100
	Confidence                float64 // 0-0.98
	PriorityScore             float64 // 0-100
	Rank                      int     // 1 = highest priority within the analysis
	Action                    ActionCode
	Reason, RecommendedAction string // filled by the Explainer
	RelatedEvent              *Event
	Evidence                  []Evidence
	Status                    string // "open" (read-only in the MVP)
}
