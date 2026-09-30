# Apply Progress: ai-energy-platform

Mode: Strict TDD (runner `go test ./... -count=1`, cwd `backend`). Store: hybrid. Delivery: stacked-to-main, `size:exception` accepted for ALL PRs (user decision 2026-09-30; review is commit by commit).

## Batch 1: Slice 0 and Slice 1 (complete)

Completed tasks: 0.1, 0.2, 1.1 - 1.16 (18/125 overall).

### Slice 0 (branch `main`)
- `master` renamed to `main`; `.gitignore` extended (`backend/*.db*`, `node_modules/`, `frontend/dist/`, `*.log`).
- Baseline commit `84dc4df chore: add baseline docs, specs and gitignore`. `.atl/` and `.claude/skills/ui-ux-pro-max/` remain ignored.

### Slice 1 (branch `feat/01-backend-foundation`, base `main`)

Commits (work units):
| Hash | Message |
|------|---------|
| f9fcf7e | build(backend): init Go module with modernc.org/sqlite v1.34.5 |
| f254974 | feat(platform): load environment configuration with redacted summary |
| 5f4cb78 | feat(domain): add entities, enums and event type normalization |
| 19c525c | feat(sqlite): open WAL database and apply embedded schema migration |
| f730355 | feat(seed): seed meters, readings and events from CSV idempotently (includes `data/` CSV copies) |
| 05c4823 | feat(api): bootstrap config, migration and seed on startup |

### TDD Cycle Evidence
| Task | RED (observed) | GREEN | REFACTOR |
|------|----------------|-------|----------|
| 1.3/1.4 config | `undefined: Load` build failure | `ok energy/internal/platform` | gofmt/vet clean |
| 1.5/1.6 event normalize | `undefined: EventDataQuality` build failure | `ok energy/internal/domain` | gofmt/vet clean |
| 1.7/1.8 migrate | `undefined: Migrate` build failure | `ok energy/internal/adapters/sqlite` | gofmt/vet clean |
| 1.9/1.10 parsers | `undefined: ParseReadings/ParseEvents` | `ok energy/internal/adapters/csvload` | gofmt/vet clean |
| 1.11/1.12 seed | `undefined: Seed, ErrDataMissing` | `ok energy/internal/adapters/csvload` | fixed generic `loadFile` signature (io.Reader) |
| 1.13 dataset seed | with `data/` moved away: `data file missing: .../readings.csv` FAIL | `ok` with data restored (12/4032/4, zero gaps, zero non-OK) | n/a |
| 1.14 bootstrap | `undefined: bootstrap` build failure | `ok energy/cmd/api` (4 tests) | n/a |
| 1.15 refactor | n/a | gofmt -l empty, go vet clean, all green | done |

### Work Unit Evidence
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./... -count=1`: ok for cmd/api, csvload, sqlite, domain, platform (5 packages) |
| Runtime harness | `DATA_DIR=../data DB_PATH=$(mktemp -d)/e.db go run ./cmd/api` run twice: run 1 `seed: meters=12 readings=4032 events=4 (inserted=4048)`, run 2 `(inserted=0)`; config log redacts password and secret |
| Rollback boundary | `backend/`, `data/`; `openspec/config.yaml` strict TDD flag |

### Task 1.16 (sdd-init refresh, manual)
No agent launch available, so `openspec/config.yaml` was updated (`strict_tdd: true`, runner `go test ./... -count=1`, cwd `backend`, vet/build/gofmt as linter/type-checker/formatter) and engram `sdd-init/prueba-tecnica` (#1) updated with the same capabilities.

### Deviations / notes
- `go.mod`: `go 1.23` plus `toolchain go1.23.4`. A bare `toolchain go1.23` is not downloadable (needs a patch version). `modernc.org/sqlite` pinned to v1.34.5 (its go directive is 1.21); `go mod tidy` keeps `go 1.23`.
- `meters` table has surrogate `id` per design DDL; test asserts no columns beyond `id`, `meter_id`, `created_at` (no name/location).
- Events whose meter has no readings are skipped with a warning (FK on `meters`); not covered by spec.
- `Seed` returns `ErrDataMissing` (wrapped, names the file) so `main` can distinguish empty-DB fatal from warn-and-continue.
- `sqlite.Migrate(ctx, db)` and `sqlite.Open(path)` return/accept `*sqlite.DB`; seed writes via `DB.WriteTx` and a `Writer` (InsertMeters/InsertReadings/InsertEvents).
- Process exits after bootstrap (no HTTP until slice 2).
- Size: slice 1 authored lines 1,463 excluding go.sum and data CSVs (584 in tests); forecast was ~420; recommend `size:exception` for PR 1 or accept as-is. See return summary.

## Batch 2: Slice 2 (complete)

Completed tasks: 2.1 - 2.17 (35/125 overall).

### Slice 2 (branch `feat/02-read-api-auth`, base `feat/01-backend-foundation`)

Commits (work units):
| Hash | Message |
|------|---------|
| 2c14ebc | feat(engine): compute per-meter hour-of-day baseline with MAD floor (2.1, 2.2) |
| 3038bec | feat(sqlite): add read repositories and app ports (2.3, 2.4, 2.5) |
| 4cd06f9 | feat(http): sign and verify stateless HMAC tokens (2.6, 2.7) |
| 064646a | feat(http): add router, auth middleware, login and health (2.8, 2.9) |
| 7bdc3ee | feat(app): serve meter catalog, detail, readings and events (2.10, 2.11) |
| 7367a3f | feat(http): expose meters, readings and events endpoints (2.12, 2.13) |
| ef09e84 | feat(dashboard): serve pre-analysis KPI summary (2.14, 2.15) |
| 28a4735 | feat(api): serve HTTP after seed with graceful shutdown (2.16, 2.17) |

### TDD Cycle Evidence (slice 2)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 2.1/2.2 baseline | `engine/baseline_test.go` | `undefined: BuildBaseline, DefaultConfig` | `ok energy/internal/engine`; one test failed first on my own wrong ratio expectation (fixed the test: per-hour ratios vary) | median/MAD, floor, daily/profile/ratio, denominator skip, insufficient history (4 cases), 30-day shift, `At` | vet/gofmt clean |
| 2.3-2.5 read repos | `adapters/sqlite/read_repos_test.go` | `undefined: NewMeterStore/NewReadingStore/NewEventStore` | ok | inclusive from/to, from-only, unknown meter, reverse-order insert, event ordering per meter | clean |
| 2.6/2.7 token | `adapters/http/auth_test.go` | `undefined: NewTokens, ErrInvalidToken` | ok | tampered payload and signature, malformed inputs, expiry boundary, restart, other secret | clean |
| 2.8/2.9 router | `adapters/http/auth_http_test.go` | `undefined: NewRouter, Deps, requireAuth` | ok | login ok/wrong pass/wrong user/empty/malformed/empty body; missing/malformed/tampered/expired/basic; panic to JSON 500; log leak check; JSON 404 | gofmt aligned publicRoutes map |
| 2.10/2.11 meter service | `app/meter_service_test.go` | `undefined: MeterService, NewMeterService, ListQuery` | ok | +109% vs +2% vs 0%, partial day ignored, insufficient history nil, filters and 4 sort keys both orders, invalid query values, detail count, readings baseline from full series | clean |
| 2.12/2.13 meters HTTP | `adapters/http/meters_http_test.go`, `testutil_test.go` | `unknown field Meters in struct literal of type Deps` | ok | q/search alias, sort, 400s (status, sort, order), 404s, 336 points, 24-point day, inverted/unparsable range, events empty array, 401s | gofmt |
| 2.14/2.15 dashboard | `app/dashboard_service_test.go`, endpoint test | `undefined: NewDashboardService, NoLatestDone` | ok | before analysis, with anomalies, clean run (confidence null), endpoint nulls present and total equals sum, 401 | gofmt |
| 2.16 wiring | `cmd/api/main_test.go` | `undefined: newHandler, serve` | ok | 401 then login then 12 meters on the real dataset; graceful shutdown on cancel | gofmt/vet |
| 2.17 refactor | n/a | n/a | gofmt -l empty, go vet clean, all packages ok, no `M-\d+` in engine non-test code | done | done |

### Work Unit Evidence (slice 2)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./... -count=1`: ok for cmd/api, csvload, adapters/http, sqlite, app, domain, engine, platform (8 packages) |
| Runtime harness | Built binary, `DATA_DIR=../data`, temp DB, `PORT=18080`: `/api/health` 200; `GET /api/meters` without token 401; login returns an 87-char token; meters 12 items (M-109 variation +110.7%, M-104 +47.4%, healthy meters -0.3/+1.1/+0.5, all status OK); `M-109` detail 200; readings 336 points with `baseline_kwh` (first 31.32, last 42.35, i.e. baseline from the first week); day range = 24 points; events for M-109 (1 UNKNOWN); dashboard `meters_count=12 total_kwh=155250.85 anomalies_count=0 high_priority_count=0` with the three nullable fields null; unknown meter 404 JSON; `status=BROKEN` 400. Graceful shutdown logged; logs contain no password or secret |
| Rollback boundary | `backend/internal/{app,adapters/http,engine/{baseline,config}.go}`, sqlite read stores in `adapters/sqlite/{meters,readings,events}.go`, `cmd/api/main.go` wiring |

### Deviations / notes (slice 2)
- HTTP package directory is `adapters/http` with package name `httpapi` (avoids clashing with `net/http`).
- Public field names follow the spec naming resolutions. Fields the specs leave open: meter detail exposes `anomaly_count` (design name); events use `timestamp` (like readings); the meter list omits the design's extra fields (`kwh_24h`, `top_severity`, ...) until a spec needs them.
- Login response also carries `user:{username}` (design), on top of the spec's `token` and `expires_at`.
- `baseline_kwh` semantics: `baseline_kwh` is 0 for a meter with insufficient history (field stays present).
- `MeterService.Readings` builds the baseline from the meter's full series even when a range is requested (a range after the baseline week still gets the right baseline).
- Method mismatch on a registered path (405) still uses the stdlib plain-text body; only unknown routes return the JSON 404. Not covered by spec.
- `DashboardService` takes an injectable `LatestDoneSource` (default `NoLatestDone`); a triangulating test already exercises the done-analysis path so slice 5 only supplies the repository.
- Size: slice 2 adds 2,854 lines across 28 files (1,459 in tests) vs ~560 forecast. `size:exception` accepted for all PRs (user decision 2026-09-30). Commits are 8 small work units.

## Batch 3: Slice 3 (complete)

Completed tasks: 3.1 - 3.17 (52/125 overall).

### Slice 3 (branch `feat/03-engine-detectors`, base `feat/02-read-api-auth`)

Commits (work units):
| Hash | Message |
|------|---------|
| 8cf62f6 | feat(engine): add detector thresholds to Config with documented defaults (3.1, 3.2) |
| c136904 | feat(engine): detect hourly consumption episodes as persistent shifts or spikes (3.3, 3.4, 3.5) |
| d5559e7 | feat(engine): report daily variation scaled by the hours each day contains (3.6, 3.7) |
| b052836 | feat(engine): detect electrical jumps and oscillation clusters (3.8, 3.9) |
| 636aefd | feat(engine): classify electrical ratio deviations as stable or unstable (3.10, 3.11) |
| 8ff127f | feat(engine): detect missing readings and non-OK status runs (3.12, 3.13) |
| 046dbf4 | feat(engine): detect hourly pattern breaks with correlation and shape (3.14, 3.15) |
| f7082fe | test(engine): guard that the engine imports only domain and stdlib (3.16) |

Detector API (all pure, take a sorted per-meter series): `DetectConsumption`, `DetectDailyVariation`, `DetectJumps` (DQ clusters + residual single jumps), `DetectRatio`, `DetectGaps`, `DetectStatus`, `DetectPatternBreaks`. All return `[]Signal` (kinds `persistent_shift|spike|daily_variation|jump|dq_cluster|ratio_stable|ratio_unstable|pattern_break|gap|status`, role seed|secondary, numeric evidence: baseline, observed, delta %, unit, counts). There is no aggregator `Detect` yet; slice 4a (`Correlate`) or 4b (`Analyze`) chains them.

### TDD Cycle Evidence (slice 3)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 3.1/3.2 config | `engine/config_test.go` | `c.MergeGapHours undefined` build failure | ok | every default asserted (15 int + 19 float fields) | gofmt |
| 3.3-3.5 hourly | `engine/detect_test.go`, `synth_test.go` | `undefined: Signal, DetectConsumption` | ok | +25% high z, +40% low z, +47%, tiny MAD 12%, 6 h persistent, 5 h spike, 3 flagged spike, 4 up/4 down, gap 2 merges / gap 3 splits, single point 70% spike / 40% dropped, evidence numbers, insufficient history, `HourlyDevPct=0.60` override | gofmt |
| 3.6/3.7 daily | `detect_test.go` | `undefined: DetectDailyVariation` | ok | +47% day, +-2% silent, -40% negative sign, partial day healthy vs tripled (+200%), insufficient history | none |
| 3.8/3.9 jumps/DQ | `quality_test.go` | `undefined: DetectJumps` | ok | V toggle 202/240, single +110% current step, ramp of 3 same-direction jumps, down/up, PF quantized, sub-threshold steps, cross-metric merge <=24 h, far apart stay separate, jump across a gap ignored | replaced pointer-into-slice with index in span builder |
| 3.10/3.11 ratio | `quality_test.go` | `undefined: DetectRatio` | ok | stable +25% x12 h, short run not standalone (+config override), unstable both signs (0.58/4.22), same-sign never unstable, PF=0 rows skipped, no baseline ratio | none |
| 3.12/3.13 gap/status | `quality_test.go` | `undefined: DetectGaps` | ok after fixing my own test helper (`hourlySeries` overwrote `Status`; `synth` now applies the mutation after building) | single gap, 3-hour run + separate run, SENSOR_FAULT run, two different statuses, clean series silent | synth helper |
| 3.14/3.15 pattern | `engine/pattern_test.go` | `undefined: DetectPatternBreaks` | ok | uniform +47% with noise, reversed day, amplified shape (corr 1) not a break, both thresholds configurable, partial day, flat profile | none |
| 3.16 purity | `engine/purity_test.go` | temporary `import _ "energy/internal/platform"` made it fail with the expected message; removed | ok | n/a | n/a |
| 3.17 refactor/verify | n/a | n/a | vet clean, gofmt empty, all 8 packages ok, no `M-\d+` / `2026-09-` in non-test engine files | n/a | done |

### Work Unit Evidence (slice 3)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./internal/engine/... -count=1`: ok; full `go test ./... -count=1`: ok for 8 packages |
| Runtime harness | N/A: pure library. Throwaway sanity run (deleted, not committed) of all detectors over `data/readings.csv`: healthy meters (101,102,103,105,107,108,110,111) yield zero seed detections; M-107 and M-110 only get single-hour stable-ratio secondaries (1 h, not standalone) |
| Rollback boundary | `backend/internal/engine/{config,detect,quality,pattern}.go` and their tests |

### Sanity counts (throwaway, per meter; pers=persistent, day=daily var, dq=DQ cluster, rS/rU=ratio stable/unstable, pat=pattern break)
- M-104 pers=1 day=4 | M-106 pers=1 day=1 jump=2 pat=1 | M-109 pers=1 day=3 jump=2 rS=1(58 h, standalone) pat=1 | M-112 dq=1 rS=1(1 h) rU=1
- Healthy: M-101, 102, 103, 105, 108, 111 zero everything; M-107 rS=2 (1 h each); M-110 rS=1 (1 h).
- No gap or status signals anywhere; no spikes anywhere.

### Deviations / notes (slice 3)
- Signal shape: flat `Signal` struct (kind, metric, start/end, sign, role, hours/flagged/count, baseline/observed/delta_pct/unit, plus `Jumps`, `Corr`, `Shape`, `Standalone`, `Detail`). No separate `Persistent` flag: the kind carries it. `DeltaPct` is in percent (47.0 = +47%).
- Task 3.10 "stable run < 6 h without consumption episode is dropped": the drop belongs to orphan promotion in slice 4a. The detector exposes `Standalone` (hours >= `RatioStandaloneMinHours`), which 4a uses; tests assert the flag and that it follows the Config value.
- Spec scenario "+25% with z 8": with the 3% MAD floor a 25% deviation has z about 5.6 at most, so the test uses that value (still above 3.5) to prove the effect gate.
- `DetectJumps` does not repeat oscillating jumps as single-jump signals; only non-oscillating jumps are emitted as `KindJump` secondaries. Jumps are measured only between readings exactly 1 h apart.
- Unstable ratio: deviant hours are grouped by `DQWindowHours` distance; stable runs by same sign and `MergeGapHours`.
- Cluster merge across metrics uses `DQWindowHours` (24) as the gap.
- Config carries fields for later slices (events, correlation, severity, confidence) so the table is complete per task 3.2.
- Size: slice 3 adds 1,805 lines across 10 files (about 950 in tests) vs ~520 forecast. `size:exception` accepted for all PRs.

## Batch 4: Slice 4a (complete)

Completed tasks: 4.1 - 4.4 (56/125 overall).

### Slice 4a (branch `feat/04a-engine-correlate-classify`, base `feat/03-engine-detectors`)

Commits (work units):
| Hash | Message |
|------|---------|
| af45fff | feat(engine): group detector signals into one incident per meter window (4.1, 4.2) |
| 7a1cc68 | feat(engine): match events to incidents with direction-aware explanatory power (4.3, 4.4) |
| 70f2ffd | refactor(engine): extract the hourly gate so recovery reuses it |
| 694b679 | feat(engine): classify incidents in order with recovery and kind assignment (4.3, 4.4) |

API (all pure): `Correlate(signals, cfg) []Incident` (any number of meters, ordered by meter then onset); `Incident{MeterID, Start, End, Sign, Persistent, Seeds, Attached}` with `IsConsumption()`, `IsDataQuality()`, `PatternOnly()`; `MatchEvent(inc, events, cfg) *EventMatch{Event, Power, Explains, Distance}`; `Recovered(inc, series, b, cfg) bool`; `Classify(inc, match, recovered) Classified{Incident, Type, Kind, Recovered, Event}`; `ClassifyIncident(inc, events, series, b, cfg)` composes the three. There is still no `Analyze` (task 4.10) and no scoring (4.6).

### TDD Cycle Evidence (slice 4a)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 4.1/4.2 correlate | `engine/correlate_test.go` | `undefined: Correlate` build failure | `ok energy/internal/engine`; one run failed first (adjacent orphan days attached to each other instead of merging), fixed by promoting a whole class before merging | one window with all evidence, two windows, secondaries never extend the window, slack boundary 2 h vs 3 h on both sides, seed merge boundary + config override, overlap uses earliest onset, orphan daily promotes, adjacent days merge, orphan pattern break, orphan jumps dropped, short vs standalone ratio, covered standalone ratio stays evidence, multi-meter ordering, incident traits per seed kind, synthetic detectors end to end, orphan jump attaches to promoted incident | none needed |
| 4.3/4.4 events | `engine/events_test.go` | `undefined: EventMatch, MatchEvent` | ok | outage on decrease/increase, UNKNOWN context, op-change both signs and spike, tolerance boundaries (0, 2, 3, 10 h) + config override, other meter ignored, DQ event window (start-2h..end), 5 raw type spellings normalized, tie-breaks (power, closeness, time) and direction-aware power | none needed |
| 4.3/4.4 classify | `engine/classify_test.go` | `undefined: Recovered, Classified, Classify, ClassifyIncident` | ok on first run | recovery 12/11/many/at series end, flagged hour and missing reading break it, config override, insufficient history; 15 ordered-classification cases; every DQ seed kind + drift; pattern-only; synthetic recovered/ongoing/no-event outages end to end | extracted `hourFlagged` from `DetectConsumption` (engine tests green before and after) |

### Work Unit Evidence (slice 4a)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./internal/engine -count=1`: ok; full `go test ./... -count=1`: ok for 8 packages; `go vet ./...` clean; `gofmt -l .` empty |
| Runtime harness | Throwaway test (deleted, not committed) running detectors, `Correlate`, `ClassifyIncident` over `data/*.csv`: 4 incidents in 12 meters: M-104 EXPLAINABLE_ANOMALY LEVEL_SHIFT_UP (OPERATIONAL_CHANGE p0.9, not recovered); M-106 FALSE_POSITIVE SUSTAINED_DECREASE (SCHEDULED_OUTAGE p1.0, recovered); M-109 REAL_ANOMALY LEVEL_SHIFT_UP (UNKNOWN p0, context only; 1 seed, 7 attached); M-112 DATA_QUALITY DATA_INCONSISTENCY (DATA_QUALITY event, 2 seeds). The 8 healthy meters produce no incident |
| Rollback boundary | `backend/internal/engine/{correlate,events,classify}.go` and their tests, plus the `hourFlagged` extraction in `detect.go` |

### Open items from slice 3 (resolved)
1. Orphan promotion drops stable-ratio signals that are not `Standalone`; the 1-2 h orphan ratio secondaries on two healthy meters therefore produce no incident (dataset run confirms). Covered by `TestOrphanStableRatioNeedsToBeStandalone`. The promoter trusts the detector's `Standalone` flag instead of re-deriving it from `Hours`, so the threshold lives in one place.
2. Ratio-unstable grouping and DQ cluster merge use `DQWindowHours` as gap: confirmed against design 2.4 (unstable = 24 h window of deviant hours; cluster windows merge "within 24 h"). No change, no new config field.

### Deviations / notes (slice 4a)
- "Within AttachSlackHours" is measured from one window end to the next window start (`start - end <= 2 h`, at most one unflagged hour between), the literal reading of design 2.5, applied identically to seed merging and secondary attachment.
- Promotion happens class by class (daily variation, standalone ratio, pattern break) with a merge between classes: consecutive orphan days form one incident and weaker evidence of the same window attaches instead of opening a second incident. Non-promotable orphans (single jumps, short ratio runs) attach to any covering incident, including promoted ones, else are dropped.
- `Incident.Persistent` is true for a persistent-shift seed and also for a promoted daily-variation seed whose window is at least `PersistMinHours` (a whole-day deviation), so an OPERATIONAL_CHANGE can explain an orphan-day incident. The design does not cover this case.
- Recovery counts only consecutive readings exactly 1 h apart starting at the hour after the window; a missing reading ends the run. A window that ends with the series is never recovered.
- `EventMatch.Power` is direction-aware (0 when type or direction does not fit); `Explains` needs `Power >= ExplainPowerMin` and `Power > 0`. DATA_QUALITY events never explain; the confidence bonus is applied by scoring (4.6) from `Event.Event.Type`.
- Event normalization is applied inside `MatchEvent` via `domain.NormalizeEventType` (idempotent), so callers may pass raw types.
- A stable-ratio drift is data quality only when the ratio run is a seed (promoted orphan); attached to a consumption incident it is corroboration only, which keeps M-109 REAL.
- Size: slice 4a adds 1,235 lines across 7 files (about 800 in tests) in 4 commits. `size:exception` accepted for all PRs.

## Batch 5: Slice 4b (complete)

Completed tasks: 4.5 - 4.15 (67/125 overall). Slice 4 is complete (4.1 - 4.15).

### Slice 4b (branch `feat/04b-engine-score-dataset`, base `feat/04a-engine-correlate-classify`)

Commits (work units):
| Hash | Message |
|------|---------|
| 024f9e5 | feat(engine): score severity, confidence and priority with evidence items (4.5, 4.6) |
| 88d1ebc | feat(engine): map type, direction and kind to a recommended action (4.7, 4.8) |
| 1dc8026 | feat(explain): render reasons and actions offline with text templates (4.7, 4.8) |
| 8f00432 | feat(engine): chain detection to ranking in Analyze with a stage hook (4.9, 4.10) |
| 519dbf8 | test(engine): validate the demo dataset outcomes, silence, shift and rename invariance (4.11, 4.12) |
| 8474ea5 | test(engine): guard the engine sources against meter and date literals (4.13) |
| (docs) | docs: document every engine threshold with its dataset evidence (4.14) and progress records |

API (all pure): `Band`, `SeverityScore(raw, type, cfg)`, `Confidence`, `ConfidenceLabel`, `Priority(sev, conf, type)`, `Rank(as)`, `Score(Classified, series, baseline, cfg) domain.Anomaly`, `Recommend(type, sign, kind)`, `Detect(series, b, cfg)`, `Analyze(Input, cfg, hook) Result`. `app.Explainer` port plus `adapters/explain.Template` (`New()`).

### TDD Cycle Evidence (slice 4b)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 4.5/4.6 score | `engine/score_test.go` | `undefined: Band, SeverityScore` build failure (a first attempt also failed on a helper name clash with `near`; helper renamed `within`) | `ok energy/internal/engine` on the first run | bands 70/69/40/39, caps 85->69 and 55->39, cap follows configured bands, confidence 1.2->0.98 and label 0.84 Medium, exact priorities 97.5/88.2/33.8/5.7, REAL beats DQ, ranking under 3 input permutations with tie-breaks, REAL consumption numbers, six-field evidence items, event clarity (explained 1.0 vs UNKNOWN 0.5), DQ score/confidence, DQ event bonus, gaps and status evidence, pattern evidence detail | none needed |
| 4.7/4.8 recommend | `engine/recommend_test.go` | `undefined: Recommend` | ok | 11 (type, sign, kind) cases, one distinct code per type | none |
| 4.7/4.8 explainer | `adapters/explain/template_test.go` | `undefined: app.Explainer` | ok on the first run | 7 action codes non-empty and distinct, reasons distinct per type, no `<no value>`/`{{`/`%!`/`<nil>`/`NaN` on empty and full anomalies, REAL reason cites 42.0/20.0/+109.4%/58 h/start and states no event explains, event kept as context, recovery time, DQ jump counts, deterministic across renders and instances, unknown action code falls back | none |
| 4.9/4.10 Analyze | `engine/engine_test.go` | `undefined: Input` build failure; first GREEN run failed on my own fixture (the shift ended before the series end, so it recovered and became FALSE_POSITIVE); fixture now runs to the series end | ok | ranked REAL then EXPLAINABLE, short series skipped, 5 identical runs, reversed input equal and caller slice untouched, hook once per stage in order, no findings, `HourlyDevPct` 0.60 keeps +100% and drops +47% | none |
| 4.11 dataset | `engine/dataset_test.go` (package `engine_test`) | Green on the first run (the engine already matched the design). Mutation check as RED evidence: `RatioStandaloneMinHours: 1` made 4 of 9 tests fail (8 anomalies, false DQ incidents on two healthy meters); config restored with `git checkout` | ok | four ranked outcomes and kinds/actions, related events, score ranges, 8 healthy meters silent, M-112 without its DQ event, +30 days, every meter renamed in reverse lexical order, summary 4/2, identical runs | none |
| 4.12 calibration | n/a | n/a | No change needed: defaults from design 2.2 pass | n/a | n/a |
| 4.13 guard | `engine/guard_test.go` | Temporary comment with a meter id and a date in `engine.go` failed the test with the expected messages; comment removed | ok | scans every non-test `*.go` (at least 10 files) | none |
| 4.14 doc | `docs/thresholds.md` | n/a | one row per Config field (35), per-meter numbers from design 2.2 and the final table | n/a | n/a |
| 4.15 verify | n/a | n/a | vet clean, gofmt empty, all 9 packages ok, `rg 'M-\d+|2026-09-' backend/internal/engine --glob '!*_test.go'` no matches, `rg expected[_]results backend` no matches | n/a | done |

### Work Unit Evidence (slice 4b)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./internal/engine -run Dataset -count=1`: ok; full `go test ./... -count=1`: ok for 9 packages |
| Runtime harness | Throwaway test (deleted, not committed) over `data/*.csv` printed the final table below |
| Rollback boundary | `backend/internal/engine/{score,recommend,engine}.go` and tests, `backend/internal/adapters/explain/`, `Explainer` in `backend/internal/app/ports.go`, `docs/thresholds.md` |

### Final dataset outcome (defaults, no calibration change)
| Rank | Meter | Type | Kind | Severity | Confidence | Priority | Action |
|------|-------|------|------|----------|------------|----------|--------|
| 1 | M-109 | REAL_ANOMALY | LEVEL_SHIFT_UP | HIGH (100.0) | 0.975 | 97.5 | INSPECT_LOAD_INCREASE |
| 2 | M-112 | DATA_QUALITY | DATA_INCONSISTENCY | HIGH (100.0) | 0.980 | 88.2 | OPEN_DQ_TICKET |
| 3 | M-104 | EXPLAINABLE_ANOMALY | LEVEL_SHIFT_UP | MEDIUM (49.0) | 0.843 | 33.7 | UPDATE_BASELINE |
| 4 | M-106 | FALSE_POSITIVE | SUSTAINED_DECREASE | LOW (34.7) | 0.858 | 5.7 | ACKNOWLEDGE_EXPECTED |

Summary text: `4 anomalies detected · 2 require priority attention` (computed in the dataset test; `app/summary.go` is task 5.6).

### Deviations / notes (slice 4b)
- Calibration (4.12): no threshold changed. M-104 confidence is 0.843 and priority 33.7 (design walk-through estimated 0.846 and 33.8); M-106 severity 34.7 (design "about 36"). Both are within the walk-through's "about" numbers.
- `Score` confidence effect term uses `HourlyDevPct` as the start and the span of the effect ramp (`(|effect| - dev) / dev`, design hardcodes 0.30 and 0.30); identical at the default, and it follows the override.
- Scores are rounded: severity and priority to 1 decimal, confidence to 3 decimals; the band is read from the rounded severity so the displayed score and band always agree.
- `Analyze` counts `MetersAnalyzed` as meters with enough history for a baseline (a skipped short series is not counted). The hook fires once per stage (baseline, detection, correlation, events, explanation, recommendation); `readings` belongs to the app. Aggregator `Detect(series, b, cfg)` is exported for the slice 5 runner.
- The explainer formats every number to 1 decimal (including power factor, so 0.9 to 0.7); `Explainer` port added to `app/ports.go`; unknown action codes fall back to a generic text.
- Summary (`app/summary.go`) stays in slice 5; the dataset test counts total and priority attention itself.
- Evidence `event` item: `Observed` is the distance to the onset in hours, `Detail` is `TYPE: description`, `EventID` set.
- Spec scenario "Override a threshold" on the full dataset: with `HourlyDevPct` 0.60 M-104 is still reported (its +47% day is an orphan daily-variation incident promoted by design 2.5, since `DailyVarPct` is 0.25), split into three incidents. The synthetic test proves the hourly gate itself; see risks in the return summary.
- Size: slice 4b adds about 2,000 lines across 12 files (about 1,050 in tests, 190 in docs) in 7 commits. `size:exception` accepted for all PRs.

## Remaining
Slices 5 - 9 (58 tasks) not started. Next: 5.1 onward (analysis jobs, summary, anomalies API, status derivation).
