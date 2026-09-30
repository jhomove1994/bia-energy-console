# Anomaly Engine Specification

## Purpose

A deterministic, pure engine (rules plus robust statistics) that turns meter readings and events into classified, scored, evidenced anomalies. Threshold values are configurable parameters documented with their derivation (see `platform-delivery`); this spec fixes behavior, not formulas.

## Requirements

### Requirement: Baseline per meter and hour of day

The engine MUST compute, per meter and hour-of-day, a robust baseline (median and MAD) from the first 7 days of that meter's series. The window MUST be relative to the series start, not a calendar date. MAD MUST have a floor relative to the median so near-zero dispersion cannot inflate scores.

#### Scenario: Window relative to data start

- GIVEN the demo dataset with every timestamp shifted by 30 days
- WHEN the engine runs
- THEN the findings are identical to the unshifted run apart from timestamps

#### Scenario: Tiny MAD does not fire

- GIVEN a healthy hour whose baseline MAD is near zero and whose value deviates 12%
- WHEN the engine evaluates it
- THEN the hour is not flagged

### Requirement: Consumption effect gate

An hour MUST be flagged only if its deviation from baseline meets BOTH the minimum effect size (default 30%) AND the minimum robust z-score (default 3.5).

#### Scenario: Both conditions met

- GIVEN an hour with +47% deviation and robust z 6
- WHEN evaluated
- THEN the hour is flagged

#### Scenario: Large z, small effect

- GIVEN an hour with +25% deviation and robust z 8
- WHEN evaluated
- THEN the hour is not flagged

#### Scenario: Large effect, low z

- GIVEN an hour with +40% deviation and robust z 2
- WHEN evaluated
- THEN the hour is not flagged

### Requirement: Persistence and spikes

Flagged hours of the same sign MUST form a persistent shift when at least the minimum run length (default 6 consecutive hours) is reached; shorter runs MUST be spikes/outliers.

#### Scenario: Persistent shift

- GIVEN 58 consecutive flagged hours of the same sign
- WHEN evaluated
- THEN one persistent shift is reported

#### Scenario: Short run is a spike

- GIVEN 3 consecutive flagged hours
- WHEN evaluated
- THEN a spike is reported, not a persistent shift

#### Scenario: Sign change breaks a run

- GIVEN 4 flagged hours above baseline followed by 4 flagged hours below baseline
- WHEN evaluated
- THEN neither run reaches the persistent minimum

### Requirement: Recovery detection

A consumption finding MUST be marked recovered when the last 12 hours of the series are back inside the effect gate; otherwise it MUST be marked ongoing.

#### Scenario: Recovered outage

- GIVEN a 12-hour drop followed by normal readings through the end of the series
- WHEN evaluated
- THEN the finding is recovered

#### Scenario: Ongoing shift

- GIVEN a shift that persists through the last reading
- WHEN evaluated
- THEN the finding is ongoing

### Requirement: Daily variation evidence

A day whose total differs from the baseline daily total by at least the daily threshold (default 25%) MUST be recorded as evidence for the overlapping finding.

#### Scenario: Daily deviation

- GIVEN a day 47% above baseline daily total
- WHEN evaluated
- THEN a daily-variation evidence item is attached to the overlapping finding

#### Scenario: Normal day

- GIVEN a day within ±2% of baseline
- WHEN evaluated
- THEN no daily-variation evidence is produced

### Requirement: Data-quality oscillation detection

The engine MUST report DATA_QUALITY evidence when voltage, current or power-factor jumps (defaults: >8 V, >50%, >=0.15) occur at least 3 times with sign reversals within 24 hours. A single step change MUST NOT qualify.

#### Scenario: Oscillating jumps

- GIVEN a meter whose voltage toggles between about 202 V and 240 V every 3 hours for a day
- WHEN evaluated
- THEN DATA_QUALITY evidence is reported

#### Scenario: Single step

- GIVEN a meter with exactly one +110% current step and no reversal
- WHEN evaluated
- THEN no DATA_QUALITY evidence is reported

#### Scenario: Same-direction jumps

- GIVEN three consecutive upward voltage jumps above the threshold with no reversal
- WHEN evaluated
- THEN no DATA_QUALITY evidence is reported

### Requirement: Gap and status detection

Missing hourly timestamps and any reading with status other than `OK` MUST be reported as DATA_QUALITY evidence. On a clean series they MUST produce nothing.

#### Scenario: Missing hour

- GIVEN a series lacking one hourly timestamp
- WHEN evaluated
- THEN a gap evidence item is reported for that timestamp

#### Scenario: Non-OK status

- GIVEN a reading with status `SENSOR_FAULT`
- WHEN evaluated
- THEN a status evidence item is reported for that reading

### Requirement: Electrical consistency ratio

The engine MUST compare `kWh / (V x A x PF / 1000)` with the meter's own median ratio. A stable shift of at least the ratio threshold (default 20%) MUST count as electrical corroboration of a consumption anomaly. An unstable ratio (large swings across hours) MUST count as data-quality evidence.

#### Scenario: Stable shifted ratio

- GIVEN a persistent consumption rise whose ratio moves +25% and stays there
- WHEN evaluated
- THEN the ratio shift is corroboration, not data-quality evidence

#### Scenario: Unstable ratio

- GIVEN a ratio swinging between 0.58 and 4.22 across hours
- WHEN evaluated
- THEN data-quality evidence is reported

### Requirement: Hourly pattern break as evidence

The engine MUST compare each day's 24-hour profile (Pearson correlation) with the meter's hour-of-day median baseline; a day below the configured correlation threshold is a pattern break. A pattern break MUST be attached as evidence to an overlapping incident and MUST NOT create a standalone anomaly unless no other finding covers that window.

#### Scenario: Break overlaps an incident

- GIVEN a day with correlation 0.54 inside a recovered outage window
- WHEN evaluated
- THEN the pattern break appears in that anomaly's evidence and no extra anomaly is created

#### Scenario: Uniform scale-up

- GIVEN a persistent +47% step with day-profile correlation 0.97
- WHEN evaluated
- THEN no pattern break is reported

#### Scenario: Isolated break

- GIVEN a day with low correlation and no other finding on that meter and window
- WHEN evaluated
- THEN one anomaly is produced for that window and classified by the ordinary rules

### Requirement: One anomaly per meter per incident window

All detections and evidence on the same meter with overlapping or adjacent windows MUST be grouped into exactly one anomaly. Disjoint windows on the same meter MUST produce separate anomalies.

#### Scenario: Several detectors, one incident

- GIVEN a meter with a persistent shift, a daily deviation, a current jump and a ratio shift in the same window
- WHEN evaluated
- THEN exactly one anomaly is produced, carrying all as evidence

#### Scenario: Two separate incidents

- GIVEN a meter with two incident windows separated by days of normal readings
- WHEN evaluated
- THEN two anomalies are produced

### Requirement: Event correlation

An event MUST be matched to an anomaly when its timestamp lies within the match window of the anomaly onset (default ±2 h). Explanatory power MUST depend on event type and direction: `SCHEDULED_OUTAGE` explains decreases only; `OPERATIONAL_CHANGE` explains persistent shifts of either sign; `DATA_QUALITY` explains no consumption change but raises data-quality confidence; `UNKNOWN` and unrecognized types explain nothing.

#### Scenario: Event within the window

- GIVEN a `SCHEDULED_OUTAGE` at 00:00 and a consumption drop starting at 00:00
- WHEN evaluated
- THEN the event is related to the anomaly and explains it

#### Scenario: Event outside the window

- GIVEN an event 10 hours before the anomaly onset
- WHEN evaluated
- THEN the event is not related

#### Scenario: Wrong direction

- GIVEN a `SCHEDULED_OUTAGE` matched to a consumption increase
- WHEN evaluated
- THEN the event does not explain the anomaly

#### Scenario: UNKNOWN explains nothing

- GIVEN an `UNKNOWN` event exactly at the onset of a +109% rise
- WHEN evaluated
- THEN the anomaly is unexplained
- AND the event may be shown as related context only

#### Scenario: DATA_QUALITY event only raises confidence

- GIVEN a DATA_QUALITY event near an oscillation anomaly
- WHEN evaluated
- THEN the anomaly's confidence is higher than without the event
- AND the classification is unchanged

### Requirement: Ordered classification

Each anomaly MUST receive exactly one type by applying rules in order: (1) oscillation, unstable ratio or gap/status evidence gives DATA_QUALITY; (2) a consumption anomaly explained by a compatible event gives FALSE_POSITIVE if recovered, else EXPLAINABLE_ANOMALY; (3) otherwise REAL_ANOMALY. DATA_QUALITY MUST be reachable from reading rules alone, without any event.

#### Scenario: Data quality without an event

- GIVEN oscillating jumps and no events for the meter
- WHEN classified
- THEN the type is DATA_QUALITY

#### Scenario: Explained and recovered

- GIVEN a drop explained by a matching outage and recovered
- WHEN classified
- THEN the type is FALSE_POSITIVE

#### Scenario: Explained and persistent

- GIVEN a persistent rise explained by an `OPERATIONAL_CHANGE`
- WHEN classified
- THEN the type is EXPLAINABLE_ANOMALY

#### Scenario: Unexplained

- GIVEN a persistent rise with no explanatory event
- WHEN classified
- THEN the type is REAL_ANOMALY

#### Scenario: Data quality takes precedence

- GIVEN a meter with both oscillation evidence and a consumption shift
- WHEN classified
- THEN the type is DATA_QUALITY

### Requirement: Severity

Each anomaly MUST have a 0-100 severity score mapped to bands (HIGH >=70, MEDIUM 40-69, LOW <40). EXPLAINABLE_ANOMALY MUST be capped at MEDIUM and FALSE_POSITIVE MUST be forced to LOW.

#### Scenario: Band boundaries

- GIVEN scores 70, 69, 40 and 39
- WHEN mapped
- THEN they are HIGH, MEDIUM, MEDIUM and LOW

#### Scenario: Explained anomaly capped

- GIVEN an EXPLAINABLE_ANOMALY whose raw score is 85
- WHEN scored
- THEN severity is MEDIUM

#### Scenario: False positive forced low

- GIVEN a FALSE_POSITIVE whose raw score is 55
- WHEN scored
- THEN severity is LOW

### Requirement: Confidence

Confidence MUST be a number in [0, 0.98], labeled High (>=0.85), Medium (0.6-0.85) or Low (<0.6).

#### Scenario: Cap

- GIVEN inputs that would compute 1.2
- WHEN scored
- THEN confidence is 0.98

#### Scenario: Label

- GIVEN confidence 0.84
- WHEN labeled
- THEN the label is Medium

### Requirement: Priority ranking

Each anomaly MUST have a priority score combining severity weight, confidence and type weight (REAL > DATA_QUALITY > EXPLAINABLE > FALSE_POSITIVE). Anomalies MUST be ranked by descending priority score, with rank 1 highest.

#### Scenario: Type weight breaks ties

- GIVEN a REAL and a DATA_QUALITY anomaly, both HIGH with equal confidence
- WHEN ranked
- THEN the REAL anomaly ranks first

### Requirement: Recommendation

Each anomaly MUST carry a non-empty recommended action appropriate to its type (inspect load for REAL, verify or replace instrumentation for DATA_QUALITY, confirm the operational change for EXPLAINABLE, no action needed for FALSE_POSITIVE).

#### Scenario: Action per type

- GIVEN one anomaly of each type
- WHEN recommendations are produced
- THEN every action is non-empty and differs by type

### Requirement: Evidence items

Each anomaly MUST list evidence items with `metric`, `baseline`, `observed`, `delta_pct`, `unit` and an optional related event.

#### Scenario: Evidence shape

- GIVEN a REAL anomaly with a consumption rise and a current rise
- WHEN evidence is produced
- THEN each item has all six fields with numeric baseline, observed and delta

### Requirement: Healthy meters are silent

Meters within their normal envelope MUST produce no anomalies.

#### Scenario: Eight healthy meters

- GIVEN the demo dataset
- WHEN the engine runs
- THEN the 8 meters without incidents (M-101, M-102, M-103, M-105, M-107, M-108, M-110, M-111) yield zero anomalies

### Requirement: Dataset validation outcomes

On the demo dataset the engine MUST yield exactly four anomalies with the outcomes below, derived only from generic rules.

#### Scenario: M-109 is REAL

- GIVEN M-109 rises about +109% from 2026-09-12 14:00 with an `UNKNOWN` event
- WHEN the engine runs
- THEN M-109 is REAL_ANOMALY, HIGH severity, priority rank 1

#### Scenario: M-106 is FALSE_POSITIVE

- GIVEN M-106 drops about 80% for 12 hours from 2026-09-08 00:00 with a matching `SCHEDULED_OUTAGE`
- WHEN the engine runs
- THEN M-106 is FALSE_POSITIVE, LOW severity, with the outage as related event

#### Scenario: M-112 is DATA_QUALITY

- GIVEN M-112 has stable daily kWh and oscillating V, PF and current on 2026-09-13/14
- WHEN the engine runs
- THEN M-112 is DATA_QUALITY, HIGH severity, even with the DATA_QUALITY event removed

#### Scenario: M-104 is EXPLAINABLE

- GIVEN M-104 steps up about +47% from 2026-09-11 00:00 with an `OPERATIONAL_CHANGE`
- WHEN the engine runs
- THEN M-104 is EXPLAINABLE_ANOMALY, MEDIUM severity

#### Scenario: Exactly four anomalies

- GIVEN the demo dataset
- WHEN the engine runs
- THEN exactly 4 anomalies exist, ranked M-109, M-112, M-104, M-106

### Requirement: No hardcoding and determinism

Engine code MUST NOT contain meter id literals, dataset-specific dates or expected results, and MUST be deterministic for identical input.

#### Scenario: Identical input

- GIVEN the same readings and events run twice
- WHEN outputs are compared
- THEN they are identical

#### Scenario: Renamed meters

- GIVEN the demo dataset with every meter id renamed
- WHEN the engine runs
- THEN types, severities and ranks match the original run

#### Scenario: Source scan

- GIVEN the engine source directory
- WHEN searched for the pattern `M-1\d\d`
- THEN there are no matches outside test fixtures

### Requirement: Configurable parameters

All thresholds MUST live in one engine configuration with documented defaults.

#### Scenario: Override a threshold

- GIVEN the hourly effect-size threshold (`HourlyDevPct`) raised from 30% to 60%
- WHEN the engine runs
- THEN the hourly detector no longer flags hourly deviations below 60%
- AND every other detector (for example daily variation, `DailyVarPct`) keeps its own threshold and is unaffected
- AND each threshold is overridable independently of the others
