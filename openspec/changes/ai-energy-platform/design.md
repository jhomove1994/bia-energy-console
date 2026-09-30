# Design: AI Energy Management Platform (MVP)

## Technical Approach

A Go 1.23 hexagonal backend holds a **pure, deterministic anomaly engine**. The engine runs rules plus robust statistics over the CSV data seeded into SQLite. An async job runner drives the engine through 7 visible stages. It persists one anomaly per meter per incident, and a template `Explainer` renders the reason and the action from numeric evidence. A React SaaS console (Vite, TanStack Query, Recharts, Tailwind) runs the demo flow. nginx serves the SPA and proxies `/api` to the Go service. `docker-compose up` runs everything with zero configuration.

The design follows the proposal and its Resolved Decisions. Deviations are listed in [Deviations from Proposal](#deviations-from-proposal).

---

## 1. Architecture

### 1.1 Layout and dependency direction

```
backend/
  go.mod                           module energy; go 1.23; toolchain go1.23.x
  cmd/api/main.go                  config -> sqlite -> migrate -> seed -> recover -> http
  internal/domain/                 entities + enums (no deps)
    meter.go reading.go event.go anomaly.go evidence.go analysis.go
  internal/engine/                 PURE: imports domain + stdlib only (no IO, no time.Now)
    config.go baseline.go detect.go quality.go pattern.go
    correlate.go events.go classify.go score.go recommend.go engine.go
  internal/app/                    use cases + ports
    ports.go meter_service.go dashboard_service.go analysis_service.go summary.go
  internal/adapters/
    sqlite/   db.go schema.sql migrate.go meters.go readings.go events.go analyses.go anomalies.go
    csvload/  readings.go events.go seed.go
    http/     router.go middleware.go auth.go handlers_*.go dto.go errors.go
    explain/  template.go
  internal/platform/config.go      env parsing
frontend/                          Vite app (section 6)
data/readings.csv data/events.csv  copies of docs/*.csv, mounted read-only
deploy/nginx.conf  backend/Dockerfile  frontend/Dockerfile  docker-compose.yml  README.md
```

```
adapters/http ─┐
adapters/sqlite├──> app (ports, services) ──> engine (pure) ──> domain
adapters/csv  ─┤                      └───────────────────────> domain
adapters/explain┘ (implements app.Explainer)
cmd/api wires everything; nothing imports adapters except cmd/api (and *_test packages).
```

| Rule | Enforcement |
|---|---|
| `engine` imports only `domain` + stdlib | review + `go list -deps` check in the engine test |
| No meter-id literal in engine | `TestEngineHasNoMeterLiterals`: scans `internal/engine/*.go` (non-test) for regex `M-\d+`, fails on a match. Comments included, so the per-meter justification lives in the README, not in `config.go`. |
| Engine takes no wall clock | the runner passes `now`; windows are relative to the first reading |

### 1.2 Domain types (`internal/domain`)

```go
type Reading struct {
    MeterID  string
    TS       time.Time // UTC, hourly
    KWh, VoltageV, CurrentA, PF float64
    Status   string    // "OK" expected
}

type EventType string // SCHEDULED_OUTAGE, MAINTENANCE, OPERATIONAL_CHANGE, DATA_QUALITY, UNKNOWN
type Event struct { ID int64; MeterID string; TS time.Time; Type EventType; Description string }

type AnomalyType string // REAL_ANOMALY, EXPLAINABLE_ANOMALY, FALSE_POSITIVE, DATA_QUALITY
type Severity    string // HIGH, MEDIUM, LOW
type AnomalyKind string // CONSUMPTION_SPIKE, CONSUMPTION_DIP, SUSTAINED_INCREASE, SUSTAINED_DECREASE,
                        // LEVEL_SHIFT_UP, LEVEL_SHIFT_DOWN, DATA_INCONSISTENCY, PATTERN_CHANGE
type ActionCode  string // INSPECT_LOAD_INCREASE, CHECK_SUPPLY_LOSS, REVIEW_TRANSIENT, OPEN_DQ_TICKET,
                        // UPDATE_BASELINE, ACKNOWLEDGE_EXPECTED, REVIEW_SCHEDULE

type Evidence struct {
    Metric    string   // consumption_kwh, daily_kwh, current_a, power_factor, voltage_v,
                       // electrical_ratio, duration_hours, voltage_jumps, current_jumps, pf_jumps,
                       // hourly_pattern, gaps, status, event
    Label     string   // human label, e.g. "Hourly consumption (window mean)"
    Baseline  *float64 // nil when not applicable (counts, events)
    Observed  float64
    DeltaPct  *float64 // (observed-baseline)/baseline*100, nil when baseline is nil or 0
    Unit      string   // kWh, kWh/h, V, A, "", h, count, corr
    Detail    string   // optional, e.g. "corr 0.53, shape deviation 36.8%"
    EventID   *int64   // set on the "event" evidence item
}

type Anomaly struct {
    ID, AnalysisID, MeterID     string
    DetectedAt                  time.Time // = WindowStart
    WindowStart, WindowEnd      time.Time
    Kind                        AnomalyKind
    Type                        AnomalyType
    Severity                    Severity
    SeverityScore               float64   // 0-100
    Confidence                  float64   // 0-0.98
    PriorityScore               float64   // 0-100
    Rank                        int       // 1 = highest priority within the analysis
    Action                      ActionCode
    Reason, RecommendedAction   string    // filled by the Explainer
    RelatedEvent                *Event
    Evidence                    []Evidence
    Status                      string    // "open" (read-only in the MVP)
}

type AnalysisStatus string // pending, running, done, failed
type Stage string          // readings, baseline, detection, correlation, events, explanation, recommendation
type Analysis struct {
    ID string; Status AnalysisStatus; Stage Stage; StageIndex, Progress int
    CreatedAt time.Time; StartedAt, FinishedAt *time.Time; Error string; Summary *Summary
}
type Summary struct {
    Total, PriorityAttention, MetersAnalyzed int
    ByType map[AnomalyType]int; BySeverity map[Severity]int
    AvgConfidence float64; Text string // "4 anomalies detected · 2 require priority attention"
}
```

### 1.3 Ports (`internal/app/ports.go`)

```go
type MeterRepository interface {
    List(ctx context.Context) ([]domain.Meter, error)
    Exists(ctx context.Context, meterID string) (bool, error)
}
type ReadingRepository interface {
    ByMeter(ctx context.Context, meterID string, from, to *time.Time) ([]domain.Reading, error)
    AllByMeter(ctx context.Context) (map[string][]domain.Reading, error) // sorted by ts
    TotalKWh(ctx context.Context) (float64, error)
}
type EventRepository interface {
    All(ctx context.Context) ([]domain.Event, error)
    ByMeter(ctx context.Context, meterID string) ([]domain.Event, error)
}
type AnalysisRepository interface {
    Create(ctx context.Context, a domain.Analysis) error
    UpdateProgress(ctx context.Context, id string, st domain.AnalysisStatus, stage domain.Stage, idx, progress int) error
    Complete(ctx context.Context, id string, finished time.Time, s domain.Summary, as []domain.Anomaly) error // one tx
    Fail(ctx context.Context, id string, finished time.Time, msg string) error
    Get(ctx context.Context, id string) (domain.Analysis, error) // ErrNotFound
    Latest(ctx context.Context) (*domain.Analysis, error)
    LatestDone(ctx context.Context) (*domain.Analysis, error)
    FailInterrupted(ctx context.Context, finished time.Time) (int, error)
}
type AnomalyRepository interface {
    List(ctx context.Context, analysisID string, f AnomalyFilter) ([]domain.Anomaly, error) // ordered by rank
    Get(ctx context.Context, id string) (domain.Anomaly, error)
}
type Explainer interface {
    Reason(a domain.Anomaly) string
    Action(a domain.Anomaly) string
}
type Clock interface{ Now() time.Time }
type IDGenerator interface{ NewID() string } // uuid v4 from crypto/rand, no dependency
```

---

## 2. Anomaly Engine

### 2.1 Pipeline and stage ownership

| # | Stage (`domain.Stage`) | Owner | Work |
|---|---|---|---|
| 1 | `readings` | app | load all readings and events from the repos, group by meter, sort |
| 2 | `baseline` | engine `BuildBaseline` | per meter, hour-of-day median/MAD for kWh, V, A, PF, ratio; daily baseline |
| 3 | `detection` | engine `Detect` | consumption flags -> episodes, daily variation, jumps -> DQ clusters, ratio runs, pattern breaks, gaps/status |
| 4 | `correlation` | engine `Correlate` | cross-signal grouping into **incidents** (1 per meter per window) |
| 5 | `events` | engine `MatchEvents` + `Classify` | event correlation, recovery, type |
| 6 | `explanation` | engine `Score` + app `Explainer.Reason` | severity, confidence, evidence, reason text |
| 7 | `recommendation` | engine `Recommend` + `Rank` + app `Explainer.Action` + `Summarize` | action code/text, priority, rank, summary, persist |

`engine.Analyze(in Input, cfg Config, hook func(domain.Stage)) Result` chains stages 2-5 plus scoring for tests. The runner calls the stage functions one by one so it can report progress.

```go
type Input  struct { Series map[string][]domain.Reading; Events []domain.Event }
type Result struct { Anomalies []domain.Anomaly; MetersAnalyzed int } // Reason/RecommendedAction empty
```

### 2.2 Config (`engine/config.go`) — final thresholds

`DefaultConfig()` returns these values. Each one is documented in the README with the same justification.

| Field | Value | Data-derived justification |
|---|---|---|
| `BaselineDays` | 7 | All incidents start on day 8 or later. The window is relative to the first reading. The median resists contamination. |
| `BaselineMinSamples` | 5 | Minimum samples per hour-of-day. Fewer means the meter is skipped (`insufficient_history`, not an anomaly). |
| `MADScale` | 1.4826 | Makes MAD consistent with σ |
| `MADFloorPct` | 0.03 | A tiny MAD made robust-z > 3.5 fire 6-33× per healthy meter |
| `HourlyDevPct` | 0.30 | Healthy max 17% (margin about 1.8×). Targets 47-130%. |
| `RobustZ` | 3.5 | Standard outlier cut. It works together with the effect gate. |
| `MergeGapHours` | 2 | Tolerates at most 2 unflagged hours inside a same-sign run |
| `PersistMinHours` | 6 | Healthy streak 0. M-106 12 h, M-109 58 h, M-104 ≥ 96 h. |
| `SpikeMinHours` | 2 | An episode under 6 h needs ≥ 2 flagged hours... |
| `SpikePointPct` | 0.60 | ...or one point ≥ 60% (healthy has no hour ≥ 40%) |
| `DailyVarPct` | 0.25 | Healthy daily ±2% |
| `PatternCorrMax` | 0.80 | Healthy min corr 0.934. Breaks: M-106 d8 0.54, M-109 d12 0.53. M-104 step 0.97. |
| `PatternShapeMin` | 0.15 | Healthy max shape deviation 5.6%. Breaks 63.7% / 36.8%. M-104 3.4%. |
| `RecoveryMinHours` | 12 | ≥ 12 consecutive in-gate hours after the window = transient (2× persistence) |
| `VoltJumpV` | 8.0 | Healthy none. M-112 about 38 V. M-109 2-3 V. |
| `CurrentJumpPct` | 0.50 | `\|ΔA\| / max(A_prev, A_cur)`, i.e. at least a 2× change. Healthy none. See deviation D2. |
| `PFJump` | 0.15 | Healthy PF range is 0.87-0.99 (width 0.12), so a healthy jump is impossible. M-112 quantized 0.58/0.72/0.98. |
| `DQMinJumps` | 3 | Jumps of one metric inside `DQWindowHours` |
| `DQMinReversals` | 2 | Sign reversals among those jumps. M-109 (1 step) and M-106 (down/up = 1 reversal) are excluded. |
| `DQWindowHours` | 24 | Oscillation look-back window |
| `RatioDevPct` | 0.20 | Healthy ratio MAD 3-5%. M-109 +23-34% stable. M-112 −46%..+294% unstable. |
| `RatioUnstableMinHours` | 3 | ≥ 3 deviant hours with **both** signs inside 24 h = instability (DQ) |
| `RatioStandaloneMinHours` | 6 | A stable ratio shift with no consumption episode seeds a DQ incident (calibration drift) |
| `EventMatchHours` | 2 | All 3 consumption events match their onset exactly (Δ = 0 h) |
| `AttachSlackHours` | 2 | Secondary signals attach to an incident within ±2 h |
| `ExplainPowerMin` | 0.7 | Minimum event power to explain a consumption incident |
| `CorrCurrentPct` / `CorrPFAbs` / `CorrRatioPct` | 0.30 / 0.10 / 0.20 | Electrical corroboration of a consumption change |
| `SevHigh` / `SevMedium` | 70 / 40 | Severity bands |
| `ConfCap` | 0.98 | Never claim certainty |
| `ConfHigh` / `ConfMedium` | 0.85 / 0.60 | Confidence labels (UI) |

### 2.3 Stage 2 — Baseline (`baseline.go`)

For each meter: `t0` = first reading ts. The baseline window is `[t0, t0 + BaselineDays·24h)`.

| Output | Algorithm |
|---|---|
| `Hour[h].Median/MAD` for kWh, V, A, PF, ratio | median and MAD of the window samples at hour-of-day `h` (UTC) |
| `SigmaKWh[h]` | `MADScale · max(MAD_h, MADFloorPct · median_h)` |
| `DailyKWh` | `Σ_h Hour[h].Median(kWh)` |
| `RatioMedian` | median of `ratio = kWh / (V·A·PF/1000)` over the window (skip rows where the denominator ≤ 0) |
| `Profile[24]` | `Hour[h].Median(kWh)`, used by the pattern detector |

### 2.4 Stage 3 — Detection (`detect.go`, `quality.go`, `pattern.go`)

Detection runs over the whole series. Each detector emits `Signal{Kind, Start, End, Sign, Role(seed|secondary), Stats}`.

| Detector | Algorithm | Role |
|---|---|---|
| Hourly flag | `dev = (kWh − med_h)/med_h`, `z = (kWh − med_h)/SigmaKWh[h]`. Flag if `\|dev\| ≥ HourlyDevPct && \|z\| ≥ RobustZ`. | input to episodes |
| Consumption episode | Scan flags chronologically. Join same-sign flags whose gap is ≤ `MergeGapHours`. `hours = end−start+1`. It is **persistent** if `hours ≥ PersistMinHours`. It is a **spike** if `hours < 6` and (`flagged ≥ SpikeMinHours` or `max\|dev\| ≥ SpikePointPct`). Otherwise it is discarded. | seed |
| Daily variation | For each calendar day: `var = (ΣkWh_day − DailyKWh)/DailyKWh`. Flag if `\|var\| ≥ DailyVarPct` (partial days are scaled by their own hours' medians). | secondary (seed only if orphaned) |
| Jumps | For consecutive hourly readings: V if `\|ΔV\| > VoltJumpV`; A if `\|ΔA\|/max(A_prev,A_cur) > CurrentJumpPct`; PF if `\|ΔPF\| ≥ PFJump`. Record `(metric, ts_cur, sign)`. | secondary |
| DQ cluster | For each metric, slide a `DQWindowHours` window over its jumps. The metric **oscillates** if the window holds `≥ DQMinJumps` jumps with `≥ DQMinReversals` sign reversals between consecutive jumps. The cluster window is `[first, last]` oscillating jump, merged across metrics when windows overlap or are within 24 h. Stats: jumps per metric, `dq_hours`. | seed |
| Ratio deviation | Hourly `rdev = (ratio − RatioMedian)/RatioMedian`. Deviant if `\|rdev\| ≥ RatioDevPct`. Group deviant hours with `MergeGapHours`. **Unstable** if a 24 h window holds ≥ `RatioUnstableMinHours` deviant hours of both signs; otherwise **stable**. | unstable → seed (DQ); stable → secondary (seed only if orphaned and ≥ `RatioStandaloneMinHours`) |
| Hourly pattern | For each complete day `d` (24 readings): `corr = pearson(kWh_d[0..23], Profile)`. `shape = mean_h \| p_d(h) − b(h) \| / b(h)`, where `p_d = kWh_d / mean(kWh_d)` and `b = Profile / mean(Profile)`. **Break** if `corr < PatternCorrMax && shape > PatternShapeMin`. Scale-invariant, so a uniform step (M-104) is not a break. | secondary (seed only if orphaned) |
| Gaps | Expected cadence 1 h. Missing timestamps form gap runs. | seed (DQ) |
| Status | Rows with `status != "OK"` form runs | seed (DQ) |

### 2.5 Stage 4 — Incident grouping (`correlate.go`) — critical

Goal: **exactly one anomaly per meter per incident window**, no matter how many raw detections fired.

```
per meter:
  seeds       = consumption episodes ∪ DQ clusters ∪ unstable ratio runs ∪ gap runs ∪ status runs
  secondaries = daily-variation days ∪ stable ratio runs ∪ single jumps ∪ pattern breaks

  1. sort seeds by Start
  2. merge seeds whose windows overlap or are within AttachSlackHours
       -> incident.Window = union of merged seed windows
       -> incident.Start  = earliest seed start (the onset)
  3. for each secondary: attach to every incident whose window overlaps
       [sec.Start - AttachSlackHours, sec.End + AttachSlackHours]
       (attaching adds evidence and corroboration, it NEVER extends the window)
  4. orphaned secondaries promote to a seed-only incident when:
       daily variation day             -> consumption incident (window = that day)
       stable ratio run ≥ RatioStandaloneMinHours -> DQ incident (drift)
       pattern break                   -> PATTERN_CHANGE incident (window = that day)
       single jumps / short ratio runs -> dropped (not enough evidence)
```

| Raw detection | How it lands |
|---|---|
| Hourly deviations | inside the consumption episode (seed) |
| Spike | its own episode (seed) |
| Persistent shift | the episode with `persistent=true` (seed) |
| DQ jumps | oscillating → DQ cluster (seed); non-oscillating → evidence on an overlapping incident |
| Ratio deviation | unstable → DQ seed; stable → corroboration evidence |
| Pattern break | evidence on an overlapping incident; standalone only if nothing else covers that day |

`Incident` carries: `Start, End, Sign` (sign of the dominant consumption episode, 0 for DQ), `Persistent`, `ConsumptionStats` (window mean obs vs baseline per metric), `DQStats`, and the attached `Signals`.

### 2.6 Stage 5 — Events and classification (`events.go`, `classify.go`)

**Event normalization**: uppercase and trim. `MAINTENANCE`, `SCHEDULED_MAINTENANCE`, and `PLANNED_OUTAGE` map to `SCHEDULED_OUTAGE` semantics. Anything unrecognized maps to `UNKNOWN`.

| Event type | Explains | Power | Match rule (same meter) |
|---|---|---|---|
| `SCHEDULED_OUTAGE` / maintenance | decreases only (`Sign < 0`) | 1.0 | `\|event.ts − incident.Start\| ≤ EventMatchHours` |
| `OPERATIONAL_CHANGE` | persistent shifts, either sign | 0.9 | same |
| `DATA_QUALITY` | nothing in consumption. Adds +0.10 to DQ confidence. | 0 | `event.ts ∈ [Start − EventMatchHours, End]` (see D3) |
| `UNKNOWN` / unrecognized | nothing | 0 | same as the consumption rule. Attached as evidence ("no explanation"). |

If several events match, pick the one with the highest power, then the one closest to the onset. The matched event is stored as `RelatedEvent` and as an `event` evidence item whatever its power.

**Recovery**: the incident is recovered (transient) if, after `End`, there are `≥ RecoveryMinHours` consecutive hours inside the hourly gate before the series ends. No free-text duration parsing.

**Classification (ordered, first match wins)**:

| # | Condition | Type |
|---|---|---|
| 1 | the incident contains a DQ cluster OR unstable ratio OR gaps/status OR stable-ratio drift | `DATA_QUALITY` |
| 2 | consumption incident with a matched event where `power ≥ ExplainPowerMin` and the direction is compatible: recovered → `FALSE_POSITIVE`; persistent → `EXPLAINABLE_ANOMALY` | FP / EXPL |
| 3 | otherwise (includes UNKNOWN events, spikes, orphan pattern changes) | `REAL_ANOMALY` |

**Kind**: DQ → `DATA_INCONSISTENCY`. Pattern-only → `PATTERN_CHANGE`. Spike → `CONSUMPTION_SPIKE`/`CONSUMPTION_DIP`. Persistent and recovered → `SUSTAINED_INCREASE/DECREASE`. Persistent and not recovered → `LEVEL_SHIFT_UP/DOWN`.

### 2.7 Stage 6 — Scoring and evidence (`score.go`)

Window statistics (consumption incidents), all computed over `[Start, End]`:
- `effect` = (Σ obs kWh − Σ hour-of-day median kWh) / Σ median.
- `dailyVar` = max |var| over the calendar days that overlap the window.
- `hours` = window length.
- `ΔA%` = mean obs A / mean baseline A − 1. `ΔPF` = mean obs PF − mean baseline PF. `Δratio%` = median ratio / RatioMedian − 1.
- `corrob` = count of {`|ΔA%| ≥ 0.30`, `|ΔPF| ≥ 0.10`, `|Δratio%| ≥ 0.20`}.

| Score | Formula |
|---|---|
| Severity (consumption) | `40·min(\|dailyVar\|,1) + 20·min(hours/24,1) + (corrob≥2 ? 20 : corrob==1 ? 10 : 0) + (type==REAL ? 20 : 0)`. Then EXPLAINABLE is capped at MEDIUM (score min(s, 69)) and FALSE_POSITIVE is forced to LOW (min(s, 39)). |
| Severity (DQ) | `50·min(jumps/10,1) + 30·min(dq_hours/24,1) + 20·min(kinds/3,1)`, where kinds ⊆ {voltage, current, pf, ratio, gaps, status} |
| Band | HIGH ≥ 70, MEDIUM ≥ 40, LOW < 40 |
| Confidence (consumption) | `0.5 + 0.2·clamp((\|effect\|−0.30)/0.30,0,1) + 0.15·min(hours/24,1) + 0.1·corrob/3 + 0.05·clarity`. `clarity` = 1 if the matched event has power ≥ 0.7 or no event exists, 0.5 if an UNKNOWN/unrecognized event matched. |
| Confidence (DQ) | `0.5 + 0.25·min(jumps/10,1) + 0.15·min(kinds/3,1) + (DATA_QUALITY event ? 0.10 : 0)` |
| Cap | `min(conf, ConfCap)` |
| Priority | `raw = sevW(H3/M2/L1) · conf · typeW(REAL 1.0, DQ 0.9, EXPL 0.6, FP 0.2)`, `PriorityScore = round(100·raw/3, 1)`. `Rank` is assigned by descending PriorityScore, ties broken by SeverityScore then MeterID (lexical, deterministic). |

**Evidence builder**: consumption incidents emit `consumption_kwh` (window mean kWh/h), `daily_kwh` (worst day vs `DailyKWh`), `current_a`, `power_factor`, `voltage_v`, `electrical_ratio`, `duration_hours`, plus `hourly_pattern` when a break is attached and `event` when an event matched. DQ incidents emit `voltage_jumps`, `current_jumps`, `pf_jumps` (count, with min/max observed in `Detail`), `electrical_ratio` (min..max in `Detail`), `duration_hours`, `gaps`/`status` when present, and `event`.

### 2.8 Stage 7 — Recommendation, explanation, summary

The engine `Recommend` maps `(Type, Sign, Kind)` to an `ActionCode`. `adapters/explain/template.go` renders text through `text/template` with numbers formatted to 1 decimal.

| Type | Condition | ActionCode | Action template |
|---|---|---|---|
| REAL | Sign > 0 | `INSPECT_LOAD_INCREASE` | "Dispatch an on-site inspection of {{meter}} within 24 h. Check for new, faulty or out-of-schedule loads (current {{A_base}}→{{A_obs}} A, PF {{pf_base}}→{{pf_obs}}) and confirm with the site operator." |
| REAL | Sign < 0 | `CHECK_SUPPLY_LOSS` | "Verify supply continuity and the meter connection for {{meter}}. Check for tripped breakers or stopped equipment." |
| REAL | spike / pattern | `REVIEW_TRANSIENT` / `REVIEW_SCHEDULE` | "Review load activity on {{meter}} during {{window}}. Escalate if it recurs." |
| DATA_QUALITY | any | `OPEN_DQ_TICKET` | "Open a metering data-quality ticket for {{meter}}. Inspect CT/PT wiring and telemetry, and exclude readings {{window}} from billing and KPIs until validated." |
| EXPLAINABLE | any | `UPDATE_BASELINE` | "Confirm with operations that '{{event.description}}' is permanent and rebaseline {{meter}}. No field action needed." |
| FALSE_POSITIVE | any | `ACKNOWLEDGE_EXPECTED` | "No action required. The deviation matches '{{event.description}}' and consumption recovered at {{recovery}}." |

Reason templates (one per type), for example REAL: "Consumption was {{effect}}% above the hour-of-day baseline for {{hours}} h from {{start}} (daily {{daily_obs}} vs {{daily_base}} kWh). Current {{ΔA}}%, power factor {{ΔPF}} and the electrical ratio corroborate a real load change. {{#event}}The linked event '{{type}}' carries no explanation.{{/event}}"

**Summary (`app/summary.go`)**:
- `Total` = count of all anomalies, including FALSE_POSITIVE.
- `PriorityAttention` = count of `Severity==HIGH && Type ∈ {REAL, DATA_QUALITY}`.
- `Text = fmt.Sprintf("%d anomalies detected · %d require priority attention", Total, PriorityAttention)`, using U+00B7. Singular forms apply for 1: "1 anomaly detected", "1 requires".

**Meter status** (from the latest done analysis): Critical = any REAL HIGH. Alert = any DATA_QUALITY, EXPLAINABLE, or non-HIGH REAL. OK = none, or only FALSE_POSITIVE.

### 2.9 Expected walk-through (dataset)

| Meter | Incident (seeds → attached) | Match / recovery | Type | Severity | Confidence | Priority |
|---|---|---|---|---|---|---|
| **M-109** | episode +, 09-12 14:00 → 09-14 23:00 (58 h, persistent) → daily var +109-110%, stable ratio +23-34%, single A jump, single PF jump, pattern break d12 (0.53/36.8%) | UNKNOWN @ onset, power 0 → no explanation. Not recovered (ends at series end). | REAL_ANOMALY, `LEVEL_SHIFT_UP` | 40·1 + 20·1 + 20 (ΔA +110%, ΔPF −0.20, ratio +29%: 3) + 20 = **100 HIGH** | 0.5+0.2+0.15+0.1+0.025 = **0.975** | 3·0.975·1.0 → **97.5, rank 1** |
| **M-112** | DQ cluster 09-13 → 09-14 (V, A, PF oscillating; about 45-48 h) → unstable ratio 0.58-4.22 | DATA_QUALITY event inside the window → +0.10 | DATA_QUALITY, `DATA_INCONSISTENCY` | 50·1 (≥ 10 jumps) + 30·1 + 20·1 (4 kinds) = **100 HIGH** | 0.5+0.25+0.15+0.10 = 1.0 → cap **0.98** | 3·0.98·0.9 → **88.2, rank 2** |
| **M-104** | episode +, 09-11 00:00 → end (≥ 96 h, persistent) → daily var +47%. Ratio stable, no pattern break (0.97/3.4%). | OPERATIONAL_CHANGE Δ 0 h, power 0.9, persistent → compatible. Not recovered. | EXPLAINABLE_ANOMALY, `LEVEL_SHIFT_UP` | 40·0.47 + 20 + 10 (ΔA +48% only) + 0 = **48.8 MEDIUM** | 0.5+0.113+0.15+0.033+0.05 = **0.846** | 2·0.846·0.6 → **33.8, rank 3** |
| **M-106** | episode −, 09-08 00:00 → 11:00 (12 h, persistent) → daily var d8, pattern break d8 (0.54/63.7%). Current down about 75%. | SCHEDULED_OUTAGE Δ 0 h, power 1.0, decrease → compatible. Recovered at 12:00 (≥ 12 h in-gate). | FALSE_POSITIVE, `SUSTAINED_DECREASE` | about 36 (≤ 39 forced) **LOW** | 0.5+0.2+0.075+0.033+0.05 = **0.858** | 1·0.858·0.2 → **5.7, rank 4** |

**Summary**: `4 anomalies detected · 2 require priority attention` (M-109 REAL HIGH, M-112 DQ HIGH).

**M-106 DQ guard**: current falls at 00:00 and rises at 12:00, which gives 2 jumps and 1 reversal (< 2), so no cluster. Intra-outage wobble (39↔61 A) stays below 50% under the symmetric denominator. **M-109 DQ guard**: 1 A jump and 1 PF jump, no V jump, so no oscillation. The ratio has a single sign, so it is stable and counts as corroboration.

**Healthy silence (8 meters)**:

| Detector | Healthy max | Threshold |
|---|---|---|
| Hourly deviation | 17% | 30% |
| Daily variation | ±2% | 25% |
| V jump | none > 8 V | 8 V |
| A jump | none > 50% (prev-denominator) | 50% symmetric (stricter) |
| PF jump | impossible (range width 0.12) | 0.15 |
| Pattern corr / shape | 0.934 / 5.6% | < 0.80 and > 15% |
| Ratio | MAD 3-5% | 20%, and orphan runs need ≥ 6 h or both signs |

Result: zero seeds and zero incidents. A dataset test asserts this.

---

## 3. Persistence (SQLite)

Driver: `modernc.org/sqlite` pinned to the latest `v1.34.x` patch whose `go.mod` declares `go ≤ 1.23`. Verify at slice 1 with `GOTOOLCHAIN=go1.23.x go mod tidy`; fall back to `v1.33.x`. Connection: `_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)`, `SetMaxOpenConns(1)` (single writer, no SQLITE_BUSY). Timestamps are stored as `TEXT` RFC3339 UTC.

```sql
-- schema.sql (embedded, applied when PRAGMA user_version < 1)
CREATE TABLE IF NOT EXISTS meters (
  id         INTEGER PRIMARY KEY,
  meter_id   TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS readings (
  id              INTEGER PRIMARY KEY,
  meter_id        TEXT NOT NULL REFERENCES meters(meter_id),
  ts              TEXT NOT NULL,
  consumption_kwh REAL NOT NULL,
  voltage_v       REAL NOT NULL,
  current_a       REAL NOT NULL,
  power_factor    REAL NOT NULL,
  status          TEXT NOT NULL,
  UNIQUE (meter_id, ts)                       -- also serves (meter_id, ts) range scans
);
CREATE TABLE IF NOT EXISTS events (
  id          INTEGER PRIMARY KEY,
  meter_id    TEXT NOT NULL REFERENCES meters(meter_id),
  ts          TEXT NOT NULL,
  type        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  UNIQUE (meter_id, ts, type)
);
CREATE TABLE IF NOT EXISTS analyses (
  id           TEXT PRIMARY KEY,
  status       TEXT NOT NULL CHECK (status IN ('pending','running','done','failed')),
  stage        TEXT NOT NULL DEFAULT '',
  stage_index  INTEGER NOT NULL DEFAULT 0,
  progress     INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  created_at   TEXT NOT NULL,
  started_at   TEXT,
  finished_at  TEXT,
  error        TEXT NOT NULL DEFAULT '',
  summary_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_analyses_status  ON analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_created ON analyses(created_at DESC);
CREATE TABLE IF NOT EXISTS anomalies (
  id                 TEXT PRIMARY KEY,
  analysis_id        TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
  meter_id           TEXT NOT NULL REFERENCES meters(meter_id),
  detected_at        TEXT NOT NULL,
  window_start       TEXT NOT NULL,
  window_end         TEXT NOT NULL,
  kind               TEXT NOT NULL,
  type               TEXT NOT NULL,
  severity           TEXT NOT NULL,
  severity_score     REAL NOT NULL,
  confidence         REAL NOT NULL,
  priority_score     REAL NOT NULL,
  rank               INTEGER NOT NULL,
  action_code        TEXT NOT NULL,
  reason             TEXT NOT NULL,
  recommended_action TEXT NOT NULL,
  related_event_id   INTEGER REFERENCES events(id),
  evidence_json      TEXT NOT NULL,
  status             TEXT NOT NULL DEFAULT 'open'
);
CREATE INDEX IF NOT EXISTS idx_anomalies_analysis_rank ON anomalies(analysis_id, rank);
CREATE INDEX IF NOT EXISTS idx_anomalies_meter         ON anomalies(meter_id);
PRAGMA user_version = 1;
```

**Seed (`csvload/seed.go`)** runs at every boot when `SEED_ON_START=true` (the default):
- One transaction, `INSERT OR IGNORE` into meters (distinct ids from readings), then readings, then events. This makes re-seeding a no-op, so it is idempotent.
- The parser accepts `2006-01-02 15:04:05`, `2006-01-02 15:04`, and RFC3339. Timezone-less values are treated as UTC.
- Malformed rows are skipped with a counted warning. The gap detector surfaces any resulting holes.
- Logs `seed: meters=N readings=N events=N (inserted=N)`.
- If the CSV is missing: the API exits non-zero when the DB is empty, and continues with a warning when it already holds data.

---

## 4. HTTP API (`adapters/http`)

Router: stdlib `http.ServeMux` with method patterns. Middleware chain: `recover → requestLog → auth (except public) → handler`. JSON only. Error shape: `{"error":{"code":"not_found","message":"meter M-999 not found"}}`.

| Method + route | Auth | Success | Errors |
|---|---|---|---|
| `GET /api/health` | public | 200 `{"status":"ok"}` | — |
| `POST /api/auth/login` | public | 200 `{token, expires_at, user:{username}}` | 400 `invalid_body`, 401 `invalid_credentials` |
| `GET /api/dashboard/summary` | ✓ | 200 `DashboardDTO` | 401 |
| `GET /api/meters?search=&status=OK\|Alert\|Critical&sort=meter_id\|variation\|kwh\|status&order=asc\|desc` | ✓ | 200 `{items: MeterRowDTO[]}` | 400 `invalid_query` |
| `GET /api/meters/{id}` | ✓ | 200 `MeterDetailDTO` | 404 |
| `GET /api/meters/{id}/readings?from=&to=` | ✓ | 200 `{meter_id, points: ReadingPointDTO[]}` | 400, 404 |
| `GET /api/meters/{id}/events` | ✓ | 200 `{items: EventDTO[]}` | 404 |
| `POST /api/ai/analyze` | ✓ | **202** `{id, status}` (same id while one is active) | 500 |
| `GET /api/ai/analysis/latest` | ✓ | 200 `AnalysisDTO` or 204 when none exist | — |
| `GET /api/ai/analysis/{id}` | ✓ | 200 `AnalysisDTO` | 404 |
| `GET /api/anomalies?analysis_id=&meter_id=&type=&severity=` | ✓ | 200 `{analysis_id, summary_text, items: AnomalyDTO[]}` (defaults to the latest done analysis; empty items if none) | 400, 404 (unknown analysis) |
| `GET /api/anomalies/{id}` | ✓ | 200 `AnomalyDTO` (with evidence and related_event) | 404 |

```jsonc
// AnomalyDTO
{ "id":"…","analysis_id":"…","meter_id":"M-109","rank":1,
  "anomaly":"Level shift up",            // human label of kind
  "kind":"LEVEL_SHIFT_UP","type":"REAL_ANOMALY","severity":"HIGH","severity_score":100,
  "confidence":0.975,"confidence_label":"High","priority_score":97.5,
  "detected_at":"…","window_start":"…","window_end":"…",
  "reason":"…","recommended_action":"…","action_code":"INSPECT_LOAD_INCREASE","status":"open",
  "related_event":{"id":3,"ts":"…","type":"UNKNOWN","description":"…","explanatory_power":0} ,
  "evidence":[ { "metric":"current_a","label":"Current (window mean)","baseline":195.2,
                 "observed":409.8,"delta_pct":109.9,"unit":"A","detail":"" } ] }
// AnalysisDTO
{ "id":"…","status":"running","stage":"detection","stage_index":3,"stage_total":7,"progress":28,
  "created_at":"…","started_at":"…","finished_at":null,"error":"",
  "summary":{"total":4,"priority_attention":2,"by_type":{…},"by_severity":{…},"avg_confidence":0.915,"text":"4 anomalies detected · 2 require priority attention"} }
// DashboardDTO
{ "meters_total":12,"total_kwh":…,"anomalies_total":4,"high_priority":2,"aggregate_confidence":0.915,
  "status_breakdown":{"ok":9,"alert":2,"critical":1},  // M-106 (FP) counts as ok
  "last_analysis":{"id":"…","status":"done","finished_at":"…"} | null,"summary_text":"…" }
// MeterRowDTO
{ "meter_id":"M-109","status":"Critical","last_reading_at":"…","last_kwh":112.4,
  "kwh_24h":2208.1,"baseline_daily_kwh":1050.3,"variation_pct":110.2,"anomaly_count":1,"top_severity":"HIGH" }
// ReadingPointDTO
{ "ts":"…","kwh":…,"voltage_v":…,"current_a":…,"power_factor":…,"status":"OK",
  "baseline":{"kwh":…,"voltage_v":…,"current_a":…,"power_factor":…} }
```

Notes:
- `variation_pct` = last 24 h kWh vs the baseline `DailyKWh`. MeterService calls `engine.BuildBaseline` (pure), so values match the engine.
- With the expected results, `status_breakdown` is `ok 9 / alert 2 / critical 1`. M-106 (FP) is OK.
- Before any analysis, status is `OK` for all meters and the anomaly counts are 0.

**Auth**:
- Env `DEMO_USER` (default `admin`), `DEMO_PASSWORD` (default `admin123`), `AUTH_SECRET` (default a dev constant plus a startup warning), `TOKEN_TTL` (default `12h`).
- Token = `base64url(json{sub,exp}) + "." + base64url(HMAC-SHA256(secret, payload))`.
- Credentials and the signature are compared with `subtle.ConstantTimeCompare`, and `exp` is checked.
- Middleware reads `Authorization: Bearer <token>`. Missing or invalid → 401 `unauthorized`.
- Public: `/api/health`, `POST /api/auth/login`.

---

## 5. Async Analysis Runner (`app/analysis_service.go`)

```
POST /analyze ─> Start(ctx)
   mu.Lock; if activeID != "" -> return activeID (single-flight)
   a = Analysis{id, pending, progress 0}; repo.Create; activeID = id; mu.Unlock
   go run(context.WithoutCancel(ctx), id)   // not tied to the HTTP request
run:
   defer recover() -> Fail(id, "internal error"); defer clear activeID
   for k, stage := range stages (1..7):
       repo.UpdateProgress(id, running, stage, k, floor((k-1)*100/7))  // 0,14,28,42,57,71,85
       sleep(StageDelay)  (select on ctx.Done)
       execute stage k   (error -> repo.Fail(id, msg); return)
   repo.Complete(id, now, summary, anomalies)   // ONE tx: insert anomalies + status=done, progress=100
```

| Aspect | Decision |
|---|---|
| Single-flight | in-memory `activeID` guarded by a mutex (single process). The DB status is the source of truth for readers. |
| Stage delay | `ANALYSIS_STAGE_DELAY_MS`: 300 in compose, 0 in tests (injected via `AnalysisService` options) |
| Atomicity | anomalies are only visible when `done`. A failed run leaves no anomalies. |
| Boot recovery | `main` calls `FailInterrupted` before serving: `UPDATE analyses SET status='failed', error='interrupted by restart', finished_at=? WHERE status IN ('pending','running')` |
| Polling | UI polls `GET /api/ai/analysis/{id}` every 500 ms while pending or running |
| History | every run is kept. Read endpoints default to the latest `done`. |

---

## 6. Frontend Architecture

Stack: Vite 5, React 18.3, TypeScript 5 (strict), React Router 6, TanStack Query 5, Recharts 2, Tailwind 3.4 (theme mapped to CSS variables), Vitest + Testing Library (minimal).

### 6.1 Routes (demo flow)

| Route | Page (container) | Purpose |
|---|---|---|
| `/login` | `LoginPage` | demo credentials hint, stores the token (localStorage) |
| `/` | `DashboardPage` | KPIs, status breakdown, summary text, top anomalies, **Run AI Analysis** |
| `/meters` | `MetersPage` | table with search, status filter, sort |
| `/meters/:meterId` | `MeterDetailPage` | 4 charts vs baseline, events, meter anomalies, **Run AI Analysis** |
| `/anomalies` | `AnomaliesPage` | ranked table, type/severity filters, summary banner |
| `/anomalies/:id` | `InvestigationPage` | reason, evidence table, window chart, related event, **Recommended action** card |

`AuthGuard` wraps every route except `/login`. On a 401 the client clears the token and redirects.

### 6.2 Structure (atomic design + container/presentational)

```
frontend/src/
  main.tsx  app/{router.tsx, providers.tsx, AuthGuard.tsx}
  api/{client.ts (fetch + bearer + ApiError), types.ts, auth.ts, meters.ts, dashboard.ts, analysis.ts, anomalies.ts}
  hooks/{useAuth.ts, useMeters.ts, useMeterReadings.ts, useDashboard.ts, useAnomalies.ts, useAnalysisJob.ts}
  components/
    atoms/      Button Card Badge Input Select Spinner ProgressBar Skeleton
    molecules/  KpiCard SeverityBadge TypeBadge StatusPill ConfidenceMeter StageStepper EvidenceRow SearchFilterBar
    organisms/  AppShell(Sidebar,Topbar) MetersTable AnomaliesTable MetricChart ChartGrid
                EvidenceTable AnalysisProgressPanel ActionCard EventTimeline
  pages/<Name>/{<Name>Page.tsx (container: hooks, routing), <Name>View.tsx (presentational: props only)}
  lib/{format.ts (kWh, %, dates), status.ts}
  styles/tokens.css  index.css
```

Rule: `*View`, atoms, molecules, and organisms receive props only. Hooks and navigation live in `*Page` containers.

### 6.3 Data fetching and polling

| Query key | Endpoint | Notes |
|---|---|---|
| `['dashboard']` | summary | staleTime 10 s |
| `['meters', params]` | meters | params in the URL search string |
| `['meter', id]`, `['readings', id]`, `['events', id]` | detail | parallel |
| `['anomalies', filters]`, `['anomaly', id]` | anomalies | |
| `['analysis', id]` | analysis | `refetchInterval: q => ['pending','running'].includes(q.state.data?.status) ? 500 : false` |

`useAnalysisJob`:
1. A `useMutation(POST /analyze)` stores the id.
2. It polls `['analysis', id]`.
3. On `done` it invalidates `dashboard`, `meters`, `anomalies`, and `meter`, then shows a CTA "View 4 anomalies" → `/anomalies`.

The `AnalysisProgressPanel` shows the `StageStepper` (7 named stages), the progress bar, and a failed state with a retry button.

### 6.4 Charts

`MetricChart` is a Recharts `ComposedChart`:
- Observed `Line` (solid, primary).
- Baseline `Line` (dashed, muted, from `point.baseline`).
- `ReferenceArea` for each anomaly window (severity color at 15% opacity).
- `ReferenceLine` at event timestamps with the type label.

`ChartGrid` renders kWh, V, A, and PF. The investigation page zooms to `window ± 24 h`.

### 6.5 Design tokens (`styles/tokens.css`)

| Token | Value | Use |
|---|---|---|
| `--bg` / `--surface` / `--border` | `#F6F8FB` / `#FFFFFF` / `#E4E9F0` | light console canvas and cards |
| `--sidebar` | `#0F1B2D` | dark navy navigation |
| `--text` / `--muted` | `#0F172A` / `#64748B` | typography |
| `--primary` | `#0EA5A4` (energy teal) | actions, observed series |
| `--accent` | `#F59E0B` (electric amber) | highlights, Run AI Analysis |
| `--sev-high` / `--sev-medium` / `--sev-low` | `#DC2626` / `#F59E0B` / `#64748B` | severity |
| `--type-real` / `--type-dq` / `--type-expl` / `--type-fp` | `#DC2626` / `#7C3AED` / `#2563EB` / `#10B981` | type badges |
| `--status-ok/alert/critical` | `#10B981` / `#F59E0B` / `#DC2626` | meter status |
| `--radius` / `--shadow` | `12px` / `0 1px 2px rgb(15 23 42 / .06)` | cards |
| font | Inter, `tabular-nums` for figures | KPIs, tables |

---

## 7. Delivery Topology

```
browser ──> web (nginx:alpine :80, host 8080)
              ├─ /        -> /usr/share/nginx/html (SPA, try_files $uri /index.html)
              └─ /api/    -> proxy_pass http://api:8080
            api (alpine runtime, :8080, not published)
              ├─ /data (./data, read-only)  CSV seed
              └─ /var/lib/energy (named volume energy-db)  energy.db
```

| File | Content |
|---|---|
| `backend/Dockerfile` | `golang:1.23-alpine` build (`CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api`) → `alpine:3.20` + `wget` for the healthcheck, non-root user |
| `frontend/Dockerfile` | `node:20-alpine` `npm ci && npm run build` → `nginx:alpine` with `deploy/nginx.conf` |
| `docker-compose.yml` | `api` (env defaults, healthcheck `wget -qO- localhost:8080/api/health`), `web` (`depends_on: api: condition: service_healthy`, ports `8080:80`), volume `energy-db` |
| env defaults | `PORT=8080 DB_PATH=/var/lib/energy/energy.db DATA_DIR=/data SEED_ON_START=true DEMO_USER=admin DEMO_PASSWORD=admin123 AUTH_SECRET=change-me-demo TOKEN_TTL=12h ANALYSIS_STAGE_DELAY_MS=300` |

Local dev: `go run ./cmd/api` (`DATA_DIR=../data`, `DB_PATH=./energy.db`), and Vite dev server with `server.proxy['/api']` → `localhost:8080`.

---

## 8. Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Engine unit (`internal/engine/*_test.go`) | each detector and rule | synthetic series builder `synth(days, profile, mutate func)`; table-driven tests |
| Engine dataset (`internal/engine/dataset_test.go`, package `engine_test`) | 4 cases + healthy silence + exact summary | loads `../../../data/*.csv` through `csvload` (test-only import) and asserts type, severity, rank, count = 4, priority attention = 2, and zero anomalies for the other 8. Asserts **no expected meter id in engine code** (the literal scan). |
| Adapters | sqlite repos, seed idempotency (seed twice → same counts), migration, `FailInterrupted` | temp-dir DB |
| App | runner: single-flight (2 starts → same id), stage order and progress values, failure → `failed` with no anomalies, atomic complete | fake repos, `StageDelay=0` |
| API (`adapters/http`, `httptest`) | login ok/401, middleware 401, meters filter/sort, 404s, analyze 202 → poll → done → anomalies 4, error shape | real sqlite temp DB + seeded fixture CSV subset, delay 0 |
| Explainer | templates render every ActionCode and type, with no `<no value>` | golden-free substring asserts |
| Frontend (cheap) | `lib/format`, `lib/status`, `useAnalysisJob` refetch predicate | Vitest |

Required synthetic RED tests (per rule):
- the 30% gate vs a 25% deviation (silent);
- 6 h persistence vs 5 h spike;
- a single step with 1 reversal is not DQ;
- 3 oscillating jumps are DQ;
- a stable ratio shift is corroboration, and both signs are DQ;
- a uniform scale-up is not a pattern break;
- a recovered decrease with an outage is FP, and a non-recovered one is EXPLAINABLE;
- an UNKNOWN event does not explain;
- a DQ event only raises confidence;
- merge: hourly flags plus a daily var plus a pattern break on the same window give **1** incident;
- a gap run and a status≠OK run give DATA_QUALITY.

Output stays short: `go test ./...` (one line per package). CI and the README show `go test ./... -count=1`.

**Strict TDD**: `openspec/config.yaml` has `strict_tdd: false` only because no runner existed yet. After slice 1 creates `go.mod` and a first test, re-run `sdd-init` so `go test ./...` becomes the runner and strict TDD (RED → GREEN → REFACTOR) applies from slice 2 on. The frontend runner (`npm test` → vitest) is added in slice 6.

---

## 9. Slice Mapping (stacked-to-main, base `main`)

Slice 0 (not counted): a baseline commit on `main` with `.gitignore`, `docs/`, and `openspec/`.

| # | Slice | Contents (tests included) | Adjustment vs proposal |
|---|---|---|---|
| 1 | Backend foundation | go.mod (pinned sqlite), domain, `platform/config`, sqlite db/schema/migrate, csvload + seed, `main` wiring, `data/` copies, seed/migrate tests | `data/` copies land here (needed by tests) |
| 2 | Read API + auth | ports, MeterService (uses `engine.BuildBaseline` + its unit tests), DashboardService (pre-analysis), http router/middleware/auth, meters/readings/events/dashboard/health/login, httptest | `baseline.go` + baseline fields of `config.go` move here from slice 3 (needed for the readings overlay and variation) |
| 3 | Engine: detectors | config, hourly flags/episodes, daily var, jumps/DQ clusters, ratio, pattern, gaps/status, synthetic tests | pattern detector included (calibrated) |
| 4 | Engine: correlate + classify + score | incident grouping, events, recovery, classify, scoring, evidence, recommend, `Analyze`, template explainer, dataset test, literal guard, `docs/thresholds` section draft | — |
| 5 | Analysis jobs + anomalies API | AnalysisService runner, analyses/anomalies repos, boot recovery, summary, 5 endpoints, meter status from the latest analysis, tests | — |
| 6 | Web shell + Dashboard | Vite/TS/Tailwind setup, tokens, AppShell, api client, auth + login, dashboard, vitest setup | — |
| 7 | Meters + Detail | meters table with filters, detail with ChartGrid vs baseline, events | — |
| 8 | Analysis + Anomalies + Investigation | useAnalysisJob, progress panel, anomalies table, investigation, ActionCard | — |
| 9 | Delivery | Dockerfiles, nginx.conf, compose, README (architecture, decisions, thresholds, run, test output) | — |

Slices 3, 4, 7, and 8 are at the budget edge. If one of them exceeds it honestly after one slicing pass, report it as `size:exception`. Do not compress the code.

---

## Architecture Decisions (ADR summary)

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Incident model | seed/secondary signals merged into one incident per meter window | one anomaly per detector hit | the brief expects 4 findings. Multiple hits become evidence, not noise. |
| Engine purity | stage functions over slices, no IO or clock | engine reading repos | deterministic, fast table tests, traceable |
| Explanation | `Explainer` port + `text/template` | LLM adapter | zero config, deterministic (Resolved Decision 2) |
| Pattern detector | Pearson + normalized shape deviation (both required) | correlation only | scale-invariant (M-104 stays normal), and the AND with shape > 15% removes noise |
| DQ discrimination | oscillation = ≥ 3 jumps and ≥ 2 reversals in 24 h per metric | any jump count | separates a step (M-109, M-106) from oscillation (M-112) |
| Recovery | measured from the readings (≥ 12 in-gate hours) | parse "for 12 hours" | robust, text-independent |
| Single-flight | in-memory mutex + DB status | DB lock/queue | a single process is enough for the MVP |
| SQLite pool | 1 connection, WAL | multi-conn | avoids SQLITE_BUSY with a background writer |
| IDs | crypto/rand UUIDv4 helper | `google/uuid` | one fewer dependency |
| Router | stdlib `ServeMux` patterns | chi/gin | Go 1.22+ makes a dependency unnecessary |
| Frontend composition | atomic design + container/presentational | page-level monoliths | user preference, testable views |

## Deviations from Proposal

| ID | Deviation | Why |
|---|---|---|
| D1 | Hourly-pattern threshold finalized: `corr < 0.80 AND shape > 15%` (was "< 0.8, to calibrate") | orchestrator calibration (healthy 0.934 / 5.6%) |
| D2 | Current jump uses the symmetric denominator `max(A_prev, A_cur)` instead of `A_prev` | stricter for healthy meters. Stops intra-outage wobble (39→61 A = +56% by prev) from creating DQ jumps on M-106. M-112 46↔218 A still qualifies (0.79). |
| D3 | A DATA_QUALITY event matches if `ts ∈ [start−2h, end]` (instead of ±2 h of onset) | the DQ cluster onset is the first detected jump, which may lag the event by up to the 3 h toggle period |
| D4 | `baseline.go` moves to slice 2 | the readings overlay and meter variation need it before the engine slices |
| D5 | Added `rank`, `action_code`, and `severity_score` columns | deterministic ordering and UI badges |
| D6 | Severity for M-112 computes to 100 (proposal said about 93) | 4 signal kinds (V, A, PF, ratio) saturate `kinds/3`. Still HIGH. |

## Threat Matrix

N/A — no shell commands, subprocesses, VCS/PR automation, executable-file classification, or process-integration boundary. HTTP routing is ordinary application routing covered by the auth middleware and API tests.

| Boundary | Applicability |
|---|---|
| Documentation-like paths | N/A: no file classification or execution |
| Git repository selection | N/A: no git automation in the product |
| Commit state | N/A |
| Push state | N/A |
| PR commands | N/A |

## Migration / Rollout

Greenfield. No migration required. The schema version is `PRAGMA user_version = 1`, and the DB file is disposable because it reseeds from CSV. Rollout is via the stacked PR slices above. Each slice can be reverted independently.

## Open Questions

- [ ] Exact `modernc.org/sqlite` patch version (verify its `go` directive ≤ 1.23 at slice 1). Non-blocking.
- [ ] Ratio-deviation healthy maximum was not profiled (only the MAD). The dataset test verifies healthy silence. If it fails, raise `RatioStandaloneMinHours` rather than `RatioDevPct`. Non-blocking.
- [ ] M-112 cluster onset timing relative to the 00:00 event is assumed to be within the window (D3 covers up to the cluster end). Non-blocking.
