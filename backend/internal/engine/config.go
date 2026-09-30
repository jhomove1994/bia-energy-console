// Package engine holds the pure anomaly-analysis logic. It imports only the
// domain package and the standard library: no IO and no wall clock.
package engine

// Config holds the tunable engine parameters. Detector thresholds are added
// alongside the detectors that use them.
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
}

// DefaultConfig returns the documented default parameters.
func DefaultConfig() Config {
	return Config{
		BaselineDays:       7,
		BaselineMinSamples: 5,
		MADScale:           1.4826,
		MADFloorPct:        0.03,
	}
}
