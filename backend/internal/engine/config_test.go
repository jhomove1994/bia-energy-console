package engine

import "testing"

func TestDefaultConfigValues(t *testing.T) {
	c := DefaultConfig()
	ints := []struct {
		name      string
		got, want int
	}{
		{"BaselineDays", c.BaselineDays, 7},
		{"BaselineMinSamples", c.BaselineMinSamples, 5},
		{"MergeGapHours", c.MergeGapHours, 2},
		{"PersistMinHours", c.PersistMinHours, 6},
		{"SpikeMinHours", c.SpikeMinHours, 2},
		{"RecoveryMinHours", c.RecoveryMinHours, 12},
		{"DQMinJumps", c.DQMinJumps, 3},
		{"DQMinReversals", c.DQMinReversals, 2},
		{"DQWindowHours", c.DQWindowHours, 24},
		{"RatioUnstableMinHours", c.RatioUnstableMinHours, 3},
		{"RatioStandaloneMinHours", c.RatioStandaloneMinHours, 6},
		{"EventMatchHours", c.EventMatchHours, 2},
		{"AttachSlackHours", c.AttachSlackHours, 2},
		{"SevHigh", c.SevHigh, 70},
		{"SevMedium", c.SevMedium, 40},
	}
	for _, tc := range ints {
		if tc.got != tc.want {
			t.Errorf("%s = %d, want %d", tc.name, tc.got, tc.want)
		}
	}
	floats := []struct {
		name      string
		got, want float64
	}{
		{"MADScale", c.MADScale, 1.4826},
		{"MADFloorPct", c.MADFloorPct, 0.03},
		{"HourlyDevPct", c.HourlyDevPct, 0.30},
		{"RobustZ", c.RobustZ, 3.5},
		{"SpikePointPct", c.SpikePointPct, 0.60},
		{"DailyVarPct", c.DailyVarPct, 0.25},
		{"PatternCorrMax", c.PatternCorrMax, 0.80},
		{"PatternShapeMin", c.PatternShapeMin, 0.15},
		{"VoltJumpV", c.VoltJumpV, 8.0},
		{"CurrentJumpPct", c.CurrentJumpPct, 0.50},
		{"PFJump", c.PFJump, 0.15},
		{"RatioDevPct", c.RatioDevPct, 0.20},
		{"ExplainPowerMin", c.ExplainPowerMin, 0.7},
		{"CorrCurrentPct", c.CorrCurrentPct, 0.30},
		{"CorrPFAbs", c.CorrPFAbs, 0.10},
		{"CorrRatioPct", c.CorrRatioPct, 0.20},
		{"ConfCap", c.ConfCap, 0.98},
		{"ConfHigh", c.ConfHigh, 0.85},
		{"ConfMedium", c.ConfMedium, 0.60},
	}
	for _, tc := range floats {
		if tc.got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}
}
