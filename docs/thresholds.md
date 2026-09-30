# Engine thresholds

Every tunable engine parameter lives in `engine.Config` (`backend/internal/engine/config.go`) and `DefaultConfig()` returns the values below. They were chosen from the demo dataset's healthy envelope and were **not** changed while calibrating: the dataset test (`engine/dataset_test.go`) passes with these defaults.

> This draft is the source for the README threshold section (slice 9). Meter ids appear only in this document and in test files, never in engine code.

## Quick reading

- **Healthy meters** (M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111) stay under every threshold, with margin.
- **The four incidents** (M-104, M-106, M-109, M-112) exceed the thresholds that concern them, by a wide margin.
- Every threshold is generic. Changing one is a config change, never a per-meter rule.

## Baseline

| Field | Default | Healthy-envelope evidence | Per-meter justification |
|-------|---------|---------------------------|-------------------------|
| `BaselineDays` | 7 | The window starts at each meter's first reading. The median resists contamination. | Every incident starts on day 8 or later (M-106 on day 8, M-104 on day 11, M-112 on day 13, M-109 on day 12), so the first week is clean. |
| `BaselineMinSamples` | 5 | 7 days give 7 samples per hour of day. | Fewer than 5 samples in an hour means the meter is skipped as `insufficient_history`, never flagged. |
| `MADScale` | 1.4826 | Standard factor that makes MAD consistent with a standard deviation. | Applies to all meters. |
| `MADFloorPct` | 0.03 | A near-zero MAD made the robust z-score exceed 3.5 between 6 and 33 times per healthy meter. | Keeps healthy meters silent; a 3% floor is still far below the 47% to 109% effects. |

## Consumption detection

| Field | Default | Healthy-envelope evidence | Per-meter justification |
|-------|---------|---------------------------|-------------------------|
| `HourlyDevPct` | 0.30 | Largest healthy hourly deviation is 17% (margin about 1.8x). | M-104 +47%, M-109 +109%, M-106 about -80%: all clear the gate. |
| `RobustZ` | 3.5 | Standard outlier cut; works with the effect gate, both must hold. | Not the deciding gate for any incident; it blocks noise on flat profiles. |
| `MergeGapHours` | 2 | Tolerates at most 2 unflagged hours inside a same-sign run. | Keeps M-109's 58 h episode as one run. |
| `PersistMinHours` | 6 | Longest healthy streak of flagged hours is 0. | M-106 12 h, M-109 58 h, M-104 96 h or more: all persistent. |
| `SpikeMinHours` | 2 | An episode under 6 h needs at least 2 flagged hours... | No spike occurs in the dataset; guards against isolated noise. |
| `SpikePointPct` | 0.60 | ...or one point of at least 60% (no healthy hour reaches 40%). | No spike occurs in the dataset. |
| `DailyVarPct` | 0.25 | Healthy daily energy varies by 2% at most. | M-104 +47%, M-109 +109% to +110%, M-106 on its outage day. |
| `PatternCorrMax` | 0.80 | Lowest healthy correlation with the baseline profile is 0.934. | Breaks: M-106 day 8 at 0.54, M-109 day 12 at 0.53. M-104's step keeps 0.97, so it is not a break. |
| `PatternShapeMin` | 0.15 | Largest healthy shape deviation is 5.6%. | Breaks: 63.7% (M-106) and 36.8% (M-109). M-104 is 3.4%. |
| `RecoveryMinHours` | 12 | Twice `PersistMinHours`. | M-106 returns to the gate for at least 12 h after its outage (recovered); M-104 and M-109 run to the end of the data (not recovered). |

## Data quality detection

| Field | Default | Healthy-envelope evidence | Per-meter justification |
|-------|---------|---------------------------|-------------------------|
| `VoltJumpV` | 8.0 | No healthy hour-to-hour voltage step above 8 V. | M-112 steps of about 38 V; M-109 only 2 to 3 V. |
| `CurrentJumpPct` | 0.50 | Steps measured against the larger reading (a 2x change at least); no healthy step. | M-112 oscillates; M-106's outage wobble (39 to 61 A) stays below 50%. |
| `PFJump` | 0.15 | Healthy power factor spans 0.87 to 0.99 (width 0.12), so a healthy jump is impossible. | M-112 quantized at 0.58, 0.72 and 0.98. |
| `DQMinJumps` | 3 | Jumps of one metric inside `DQWindowHours`. | M-112 has at least 10. |
| `DQMinReversals` | 2 | Sign reversals among those jumps. | M-109 (1 step) and M-106 (down then up, 1 reversal) are excluded. |
| `DQWindowHours` | 24 | Look-back window for oscillation and for merging cluster windows. | M-112's cluster spans about 45 to 48 h. |
| `RatioDevPct` | 0.20 | Healthy electrical ratio MAD is 3% to 5%. | M-109 stable at +23% to +34%; M-112 unstable from -46% to +294%. |
| `RatioUnstableMinHours` | 3 | At least 3 deviant hours of both signs inside 24 h mean instability. | M-112 (ratio 0.58 to 4.22). |
| `RatioStandaloneMinHours` | 6 | A stable ratio run shorter than 6 h cannot seed an incident: the healthy meters M-107 and M-110 have 1 to 2 hour runs. | M-109's 58 h drift is covered by its consumption episode, so it stays corroboration. Lowering this to 1 makes M-107 and M-110 report false data-quality incidents (checked). |

## Correlation and events

| Field | Default | Healthy-envelope evidence | Per-meter justification |
|-------|---------|---------------------------|-------------------------|
| `EventMatchHours` | 2 | Tolerance between an event and the incident onset. | All three consumption events match their onset exactly (0 h). |
| `AttachSlackHours` | 2 | Secondary signals attach within 2 h of the window and seeds within 2 h merge. | Single current and power factor jumps of M-109 attach to its episode. |
| `ExplainPowerMin` | 0.7 | An explaining event needs power of at least 0.7 (outage 1.0 on a decrease, operational change 0.9 on a persistent shift). | M-106 (outage, 1.0) and M-104 (operational change, 0.9) are explained; M-109's `UNKNOWN` event has power 0. |
| `CorrCurrentPct` | 0.30 | Electrical corroboration of a consumption change: current. | M-109 +110%, M-104 +48%, M-106 about -75%. |
| `CorrPFAbs` | 0.10 | Corroboration: power factor moves by at least 0.10. | M-109 -0.20. |
| `CorrRatioPct` | 0.20 | Corroboration: ratio moves by at least 20%. | M-109 +29%. |

## Severity and confidence

| Field | Default | Healthy-envelope evidence | Per-meter justification |
|-------|---------|---------------------------|-------------------------|
| `SevHigh` | 70 | Severity band boundary (HIGH from 70). | M-109 and M-112 score 100. |
| `SevMedium` | 40 | Severity band boundary (MEDIUM from 40, LOW below). | M-104 scores 49.0 (MEDIUM, and an explained anomaly can never exceed 69); M-106 scores 34.7 and is forced LOW. |
| `ConfCap` | 0.98 | Never claim certainty. | M-112 computes 1.0 and is reported at 0.98. |
| `ConfHigh` | 0.85 | Confidence label High from 0.85. | M-109 0.975 and M-112 0.98 are High; M-104 0.843 is Medium. |
| `ConfMedium` | 0.60 | Confidence label Medium from 0.60, Low below. | No incident falls below it. |

## Final dataset outcome (defaults)

| Rank | Meter | Type | Severity | Confidence | Priority |
|------|-------|------|----------|------------|----------|
| 1 | M-109 | REAL_ANOMALY | HIGH (100) | 0.975 | 97.5 |
| 2 | M-112 | DATA_QUALITY | HIGH (100) | 0.98 | 88.2 |
| 3 | M-104 | EXPLAINABLE_ANOMALY | MEDIUM (49.0) | 0.843 | 33.7 |
| 4 | M-106 | FALSE_POSITIVE | LOW (34.7, forced LOW) | 0.858 | 5.7 |

Summary: `4 anomalies detected · 2 require priority attention`.

## Verify

- `cd backend && go test ./internal/engine -run Dataset -count=1` proves the four outcomes, the eight silent meters, the 30-day shift and the meter renaming.
- `TestEngineHasNoMeterLiterals` proves the engine sources never name a meter or a dataset date.
