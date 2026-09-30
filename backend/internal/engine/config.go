// Package engine holds the pure anomaly-analysis logic. It imports only the
// domain package and the standard library: no IO and no wall clock.
package engine

// Config holds every tunable engine parameter. Detectors read thresholds only
// from here; DefaultConfig documents the value of each one.
type Config struct {
	// BaselineDays is the length of the baseline window, counted from the
	// first reading of each meter.
	BaselineDays int
	// BaselineMinSamples is the minimum number of samples per hour of day; a
	// meter with fewer is reported as having insufficient history.
	BaselineMinSamples int
	// MADScale makes the MAD consistent with a standard deviation.
	MADScale float64
	// MADFloorPct floors the MAD at this fraction of the median so a near-zero
	// dispersion cannot inflate robust z-scores.
	MADFloorPct float64

	// HourlyDevPct is the minimum relative deviation from the hour-of-day
	// median for an hour to be flagged (effect gate).
	HourlyDevPct float64
	// RobustZ is the minimum robust z-score for an hour to be flagged. It works
	// together with HourlyDevPct: both must hold.
	RobustZ float64
	// MergeGapHours is the number of unflagged hours tolerated inside a
	// same-sign run.
	MergeGapHours int
	// PersistMinHours is the run length from which a shift is persistent.
	PersistMinHours int
	// SpikeMinHours is the number of flagged hours that makes a run shorter
	// than PersistMinHours a spike.
	SpikeMinHours int
	// SpikePointPct makes a single point a spike when its deviation reaches it.
	SpikePointPct float64
	// DailyVarPct is the minimum relative deviation of a day's energy from its
	// baseline to be reported.
	DailyVarPct float64
	// PatternCorrMax and PatternShapeMin define an hourly pattern break: the
	// day's correlation with the baseline profile is below PatternCorrMax and
	// its normalized shape deviation is above PatternShapeMin.
	PatternCorrMax  float64
	PatternShapeMin float64
	// RecoveryMinHours is the number of consecutive in-gate hours after a
	// window that makes it transient.
	RecoveryMinHours int

	// VoltJumpV is the voltage step between consecutive hours, in volts, that
	// counts as a jump.
	VoltJumpV float64
	// CurrentJumpPct is the current step relative to the larger of the two
	// consecutive readings that counts as a jump.
	CurrentJumpPct float64
	// PFJump is the absolute power-factor step that counts as a jump.
	PFJump float64
	// DQMinJumps and DQMinReversals define an oscillating metric: at least
	// DQMinJumps jumps with DQMinReversals sign reversals inside DQWindowHours.
	DQMinJumps     int
	DQMinReversals int
	DQWindowHours  int

	// RatioDevPct is the minimum relative deviation of the electrical ratio
	// from its baseline median for an hour to be deviant.
	RatioDevPct float64
	// RatioUnstableMinHours is the number of deviant hours of both signs
	// inside DQWindowHours that makes the ratio unstable.
	RatioUnstableMinHours int
	// RatioStandaloneMinHours is the run length from which a stable ratio
	// shift may seed an incident on its own.
	RatioStandaloneMinHours int

	// EventMatchHours is the tolerance between an incident onset and an event.
	EventMatchHours int
	// AttachSlackHours is the tolerance for attaching secondary signals to an
	// incident window.
	AttachSlackHours int
	// ExplainPowerMin is the minimum event power to explain a consumption
	// incident.
	ExplainPowerMin float64
	// CorrCurrentPct, CorrPFAbs and CorrRatioPct are the electrical
	// corroboration thresholds of a consumption change.
	CorrCurrentPct float64
	CorrPFAbs      float64
	CorrRatioPct   float64

	// SevHigh and SevMedium are the severity score bands.
	SevHigh   int
	SevMedium int
	// ConfCap is the maximum confidence ever reported.
	ConfCap float64
	// ConfHigh and ConfMedium are the confidence label cut points.
	ConfHigh   float64
	ConfMedium float64
}

// DefaultConfig returns the documented default parameters.
func DefaultConfig() Config {
	return Config{
		BaselineDays:       7,
		BaselineMinSamples: 5,
		MADScale:           1.4826,
		MADFloorPct:        0.03,

		HourlyDevPct:     0.30,
		RobustZ:          3.5,
		MergeGapHours:    2,
		PersistMinHours:  6,
		SpikeMinHours:    2,
		SpikePointPct:    0.60,
		DailyVarPct:      0.25,
		PatternCorrMax:   0.80,
		PatternShapeMin:  0.15,
		RecoveryMinHours: 12,

		VoltJumpV:      8.0,
		CurrentJumpPct: 0.50,
		PFJump:         0.15,
		DQMinJumps:     3,
		DQMinReversals: 2,
		DQWindowHours:  24,

		RatioDevPct:             0.20,
		RatioUnstableMinHours:   3,
		RatioStandaloneMinHours: 6,

		EventMatchHours:  2,
		AttachSlackHours: 2,
		ExplainPowerMin:  0.7,
		CorrCurrentPct:   0.30,
		CorrPFAbs:        0.10,
		CorrRatioPct:     0.20,

		SevHigh:    70,
		SevMedium:  40,
		ConfCap:    0.98,
		ConfHigh:   0.85,
		ConfMedium: 0.60,
	}
}
