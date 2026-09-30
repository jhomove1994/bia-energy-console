# Apply Progress: ai-energy-platform

Mode: Strict TDD (backend runner `go test ./... -count=1`, cwd `backend`; frontend runner `npx vitest run --reporter=dot`, cwd `frontend`). Store: hybrid. Delivery: stacked-to-main, `size:exception` accepted for ALL PRs (user decision 2026-09-30; review is commit by commit).

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
| 07b4d3c | docs: document every engine threshold with its dataset evidence (4.14) |
| eb85725 | docs(openspec): mark slice 4b complete and record apply progress |

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

## Batch 6: Slice 5 (complete)

Completed tasks: 5.1 - 5.14 (81/125 overall).

### Slice 5 (branch `feat/05-analysis-jobs-api`, base `feat/04b-engine-score-dataset`)

Commits (work units):
| Hash | Message |
|------|---------|
| 1216e85 | docs(openspec): clarify threshold override scenario per detector (spec wording fix + `docs/thresholds.md` note) |
| 9681eb3 | feat(platform): generate v4 ids and declare analysis and anomaly ports (5.1, 5.2) |
| 194dd70 | feat(sqlite): persist analyses and anomalies with atomic completion (5.3, 5.4) |
| 24b3eb7 | feat(app): summarize anomalies with priority attention and headline text (5.5, 5.6) |
| ad3204e | feat(app): run analyses as a single-flight background job with staged progress (5.7, 5.8) |
| ffadeda | feat(app): derive meter status from the latest done analysis (5.9, 5.10) |
| a3d4488 | feat(app): serve anomaly list and detail from the latest done analysis (5.11, 5.12) |
| 30eb947 | feat(http): expose analysis jobs and anomalies endpoints (5.11, 5.12) |
| 4c3df39 | feat(api): recover interrupted analyses at boot and wire the analysis stack (5.13) |

### TDD Cycle Evidence (slice 5)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 5.1/5.2 uuid | `platform/uuid_test.go` | `undefined: NewID, UUIDGenerator` build failure | ok | v4 format x50, 1,000 unique, generator port | none |
| 5.3/5.4 stores | `sqlite/analyses_test.go` | `undefined: AnomalyStore, NewAnalysisStore` build failure | ok | lifecycle + started_at stability, atomic Complete (FK failure on second anomaly leaves analysis running and zero anomalies), Fail, ErrNotFound, Latest vs LatestDone with failed/running, FailInterrupted (2 of 4 rows, idempotent), rank order and 4 filters, evidence and event round trip | gofmt |
| 5.5/5.6 summary | `app/summary_test.go` | `undefined: Summarize` | ok | demo shape, zero, singular forms, FALSE_POSITIVE only in total, HIGH EXPLAINABLE not priority, by-type/severity, mean 0.75 | none |
| 5.7/5.8 runner | `app/analysis_service_test.go` | `undefined: AnalysisService` build failure | ok on the first run; `go test -race ./internal/app/...` clean | 8 concurrent Starts share one id and one row, new id after done, retry after engine error and after panic, 7 stages with progress 0,14,28,42,57,71,85 then 100, done only at 100, results visible only at Complete, explainer + summary persisted, real engine finds the shifted meter, readings error fails, 25 ms delay shows at least two stages | none |
| 5.9/5.10 status | `app/meter_status_test.go`, `dashboard_service_test.go` | `undefined: DeriveStatuses` | ok | 8 derivation cases (Critical only for REAL HIGH; DQ/EXPLAINABLE HIGH stay Alert; FP only stays OK), per-meter grouping, latest-done source, repository error, mean confidence 0.75, failed newer run keeps A, newer done C replaces | none |
| 5.11/5.12 service | `app/anomaly_service_test.go` | `undefined: AnomalyService` | ok | default latest done, none yet (empty non-nil), explicit id, 3 filters, failed run exposes nothing, unknown analysis and bad filters, Get | none |
| 5.11/5.12 HTTP | `adapters/http/analysis_http_test.go` | `unknown field Analysis in struct literal of type Deps` build failure; first GREEN run failed on my own assertions (unit expected on `electrical_ratio` evidence; a run left in flight closed the DB) and both were fixed in the test | ok | demo dataset from `data/*.csv`: 202 `{id}` exact body, repeat while active same id, progress monotonic with at least two stages, nothing visible before done, new id after done, done payload, 404, 204, empty list before analysis, 4 ranked rows, 3 filters, 404/400s, detail with evidence and UNKNOWN event, meters Critical/Alert/OK, dashboard, five routes 401 | none |
| 5.13 boot | `cmd/api/main_test.go` | `stuck-pending after boot = ... Status:pending ... want failed/interrupted` | ok | pending and running rows both failed, log line, new run reaches done | none |
| 5.14 verify | n/a | n/a | vet clean, gofmt empty, all 9 packages ok, race clean on `internal/app`, no `M-\d+` in non-test backend code | n/a | done |

### Work Unit Evidence (slice 5)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go test ./... -count=1`: ok for 9 packages; `go test -race ./internal/app/... -count=1`: ok |
| Runtime harness | Built binary, `DATA_DIR=../data`, temp DB, `ANALYSIS_STAGE_DELAY_MS=300`, `PORT=18085`: login (87-char token); `POST /api/ai/analyze` 202 id `7a4f...`; second POST 202 with the same id; polling showed running readings 1/0, baseline 2/14, detection 3/28, correlation 4/42, events 5/57, explanation 6/71, then done recommendation 7/100 (empty error); `GET /api/anomalies` summary `4 anomalies detected · 2 require priority attention`, ranks 1-4 M-109 REAL HIGH, M-112 DATA_QUALITY HIGH, M-104 EXPLAINABLE MEDIUM, M-106 FALSE_POSITIVE LOW; M-109 detail evidence metrics consumption_kwh, daily_kwh, current_a, power_factor, voltage_v, electrical_ratio, duration_hours, hourly_pattern, event with reason (92.8 vs 44.1 kWh/h, +110.5%), action and UNKNOWN related event; meters M-109 Critical, M-104 and M-112 Alert, 9 OK; dashboard 12 meters, 4 anomalies, 2 high, aggregate confidence 0.914, last analysis done; graceful shutdown |
| Rollback boundary | `backend/internal/{platform/uuid.go,app/{analysis_service,anomaly_service,summary}.go,adapters/sqlite/{analyses,anomalies}.go,adapters/http/{handlers_analysis,handlers_anomalies}.go}`, the `AnalysisStatuses` block in `app/meter_service.go`, the new ports in `app/ports.go`, router `Deps`, `cmd/api/main.go` wiring and boot recovery |

### Deviations / notes (slice 5)
- Spec wording fix applied (commit 1216e85): the "Override a threshold" scenario now states that raising `HourlyDevPct` only changes the hourly detector and every threshold is overridable independently; `docs/thresholds.md` carries the same note. No engine code changed.
- Added `app/anomaly_service.go` (`AnomalyService`, `AnomalyQuery`, `AnomalyList`) so the list defaults, filter validation and 404 rules live in the app layer instead of the handlers (not named in the tasks). It exposes anomalies only for `done` analyses.
- The `explanation` and `recommendation` stages are progress markers driven by the engine hook; the Explainer runs after `engine.Analyze` returns (it needs the scored anomaly and its action code) and before the single `Complete` transaction. The `readings` stage belongs to the runner.
- `AnalysisService` options: `WithStageDelay`, `WithClock`, `WithAnalyzer` (test seam for engine errors, panics and pauses; the default wraps `engine.Analyze`), `WithLogger`. The single-flight slot is released under the same lock as the terminal write, so a client that sees `done` can start the next run at once.
- `NoAnalysis` and `NoLatestDone` stay in the code for tests and fixtures; `main` wires the repository-backed sources (`NewAnalysisStatuses`, the analysis store as `LatestDoneSource`).
- API shape follows the specs: analyze returns exactly `{id}`; analysis has `stage_index`, `stage_total` (7) and a `summary` (`total`, `priority_attention`, `meters_analyzed`, `by_type`, `by_severity`, `avg_confidence`, `text`) only when done; anomalies carry `priority_rank`, `confidence_label`, `anomaly` (kind label), nullable `related_event` (id, timestamp, type, description; no `explanatory_power`) and `evidence[]`. `GET /api/anomalies` returns `analysis_id` and `summary_text` as null when no analysis is done. `router.Deps.Engine` (optional) supplies the confidence label bands.
- `AvgConfidence` is rounded to three decimals (demo 0.914).
- `started_at` is stamped by the store on the first `running` update (the port carries no timestamp), using an injectable clock defaulting to `time.Now`.
- Size: 23 files, 2,717 additions and 6 deletions in 9 commits (about 1,500 in tests). `size:exception` accepted for all PRs.

## Batch 7: Slice 6 (complete)

Completed tasks: 6.1 - 6.13 (94/125 overall).

### Slice 6 (branch `feat/06-web-shell-dashboard`, base `feat/05-analysis-jobs-api`)

Commits (work units):
| Hash | Message |
|------|---------|
| 84a106f | build(frontend): scaffold Vite, React, Tailwind and Vitest toolchain (6.1) |
| 25be719 | feat(frontend): add display formatters and status token helpers (6.2, 6.3) |
| f4671f2 | feat(frontend): add design tokens, Tailwind theme and atoms (6.4, 6.5) |
| be322bb | feat(frontend): add KPI, badge, status and confidence molecules (6.6) |
| 1c99a0f | feat(frontend): add API client with bearer auth, typed errors and 401 handling (6.7) |
| 22df766 | feat(frontend): add auth session, API modules and route guard (6.8, 6.9) |
| 8a21351 | feat(frontend): add AppShell organism with sidebar, topbar and skip link (6.10) |
| fbbfa8b | feat(frontend): add login page with inline validation and error feedback (6.11) |
| 0c2c351 | feat(frontend): add dashboard KPIs with empty, loading, error and run-analysis states (6.12) |
| 4d7342f | feat(frontend): wire routes, app layout and session username (6.9, 6.12) |
| 654eb7c | style(frontend): mute empty KPI placeholders and size the confidence figure |

### Design system (ui-ux-pro-max, one coherent system)
Queries run: `--design-system` for "energy management saas dashboard" (returned a dark glassmorphism/Fira direction) and for "energy utilities dashboard data-dense" with dials density 8, motion 2, variance 3 (returned Minimalism and Swiss style, navy/grey corporate, amber accent), plus `--domain typography|chart|ux` and `--stack react`.
Decision: keep the design section 6.5 palette (it matches the second result: light Swiss-style data console, navy chrome, restrained amber) and harden it for accessibility. The dark glassmorphism suggestion was rejected because it hurts dense-table legibility and contrast.
- Surfaces: canvas `#F6F8FB`, surface `#FFFFFF`, border `#E4E9F0`, text `#0F172A` (16.8:1), muted `#5B6B80` (5.1:1 on canvas; design's `#64748B` measured 4.47:1, so darkened).
- Navigation: sidebar `#0F1B2D`, hover `#16263D`, text `#CBD5E1` (11.6:1), active item has an amber inset bar.
- Brand: primary teal `#0EA5A4` for series, meters and decoration only (3.0:1 on white, not for text); `--primary-strong` `#0F766E` for buttons, links, focus ring (5.5:1 with white); accent amber `#F59E0B` with `#0F172A` text (8.3:1) for Run AI Analysis.
- Severity/type/status share a base color plus an "-ink" shade for text on a 10% tint (all 4.6:1 or better): red `#DC2626`/`#B91C1C`, amber `#F59E0B`/`#B45309`, slate `#64748B`/`#475569`, violet `#7C3AED`/`#6D28D9`, blue `#2563EB`/`#1D4ED8`, green `#10B981`/`#047857`. Badges always carry text (and a dot for severity/status), never color alone.
- Typography: Inter Variable, self-hosted via `@fontsource-variable/inter` (no CDN, works offline in Docker); `tabular-nums` for every figure (`.tabular`); KPI figure 30px semibold, label 12px uppercase tracking-wide, body 14px.
- Shape and motion: radius 12px, card shadow `0 1px 2px rgb(15 23 42 / .06)`, 150ms ease `cubic-bezier(.2,0,0,1)`, `prefers-reduced-motion` zeroes durations and disables spin/pulse; visible 2px focus ring, skip link, `role=alert` for errors, `role=meter/progressbar` semantics.
- Tokens live in `src/styles/tokens.css` as RGB channel triplets (hex noted beside each) and are mapped in `tailwind.config.ts` so Tailwind opacity modifiers work; no raw hex inside components.

### TDD Cycle Evidence (slice 6)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 6.2/6.3 | `lib/format.test.ts`, `lib/status.test.ts` | `Failed to resolve import "./format"` / `"./status"` (2 files failed, no tests) | 12 passed after implementation; first run had 2 failures (`Sept` vs `Sep` from en-GB ICU) fixed in the code by composing the month from en-US | thousands and padding, plus/minus/zero signs, ratios 0.914 and 0.75, dates and missing values, every enum value | none |
| 6.5/6.6 | `components/molecules/molecules.test.tsx` | `Failed to resolve import "./ConfidenceMeter"` | 23 total passed | KPI label/value/caption and group role, 3 severities, type, 3 statuses, meter with value and null | none |
| 6.7 | `api/client.test.ts` | `Failed to resolve import "./client"` | 9 passed | bearer present/absent, JSON body, 204, error body to ApiError, non-JSON error body, network failure, 401 clears token and calls handler, 401 with opt-out leaves the session | none |
| 6.8/6.9 | `hooks/useAuth.test.tsx`, `app/AuthGuard.test.tsx` | `Failed to resolve import "../app/providers"` / `"./AuthGuard"` | 41 total passed | signed out/in at start, login stores token and posts credentials, wrong credentials keep signed out, logout, 401 signs out, guard redirects root and deep links, guard lets a token through | removed a stray re-export in `api/types.ts` |
| 6.10 | `components/organisms/AppShell.test.tsx` | `Failed to resolve import "./AppShell"` | passed | content and user, aria-current on the active route, logout callback, skip link and main landmark | none |
| 6.11 | `pages/Login/LoginView.test.tsx`, `LoginPage.test.tsx` | `Failed to resolve import "./LoginView"` / `"./LoginPage"` | 54 total passed | submit values, empty-field inline errors without submit, announced error, disabled while submitting, hint; page: success lands on dashboard, wrong credentials stay with alert, network error message, already signed in | none |
| 6.12 (+6.9 wiring) | `pages/Dashboard/DashboardPage.test.tsx`, `app/router.test.tsx`, `useAuth.test.tsx` (username cases) | `Failed to resolve import "./DashboardPage"` / `"./router"`; 2 username assertions failed (`toHaveTextContent`) | 68 passed | KPIs after analysis (12, 12,345.7 kWh, 4, 2, 91%, date), summary text, pre-analysis nulls with empty state and "across 12 meters", loading status, error then Retry recovers, run refreshes KPIs, trigger disabled in flight, start failure message; routes: anonymous to login, login to dashboard in shell, sign out, 401 mid-session to login, unknown path to dashboard | dashboard test helpers simplified (`mockApi` with Error entries); router future flags adopted to silence warnings; `act` around the in-flight release |
| 6.13 verify | n/a | n/a | tsc clean, 68 tests, build ok, go tests ok | n/a | done |

Test summary: 68 tests in 11 files; 0 stderr warnings after cleanup. No CSS-class assertions (behavioral queries by role and text only).

### Work Unit Evidence (slice 6)
| Evidence | Value |
|---|---|
| Focused test command | `cd frontend && npx vitest run --reporter=dot`: 11 files, 68 tests passed; `npx tsc --noEmit` clean; `npm run build`: JS 225.09 kB (71.57 kB gzip), CSS 22.72 kB (5.40 kB gzip), Inter woff2 subsets; `cd backend && go test ./... -count=1`: 9 packages ok |
| Runtime harness | Built API binary with `DATA_DIR=../data`, temp DB, `PORT=18085`, `ANALYSIS_STAGE_DELAY_MS=300`, demo user env; `API_TARGET=http://localhost:18085 npx vite --port 15173`. curl: `/` 200 index.html, `/src/main.tsx` 200, `/api/health` through the proxy `{"status":"ok"}`, login through the proxy returned a token, dashboard without a token 401, with token pre-analysis `meters_count 12, total_kwh 155250.85`, nulls. Headless Chrome (CDP script in the scratchpad, no repo files) rendered login, dashboard pre-analysis with empty state, Run AI Analysis click showing the "Analysis started" note, then post-analysis KPIs 12 / 155,250.9 kWh / 4 / 2 / 91% and banner `4 anomalies detected · 2 require priority attention`; mobile (390px) stacks cleanly. Both servers stopped. |
| Rollback boundary | `frontend/` (whole directory); no backend files touched |

### Deviations / notes (slice 6)
- Recharts is not installed in this slice (no chart yet); slice 7 adds it. `@fontsource-variable/inter` was added (self-hosted font; not in the task list). `@types/node` added as a dev dependency for `process.env` in `vite.config.ts`.
- `vite.config.ts` reads `API_TARGET` (default `http://localhost:8080`) so the proxy target can be overridden without editing config.
- Test type/globals: Vitest `globals: true` with `@testing-library/jest-dom/vitest` in `src/test/setup.ts`, which also clears localStorage after each test.
- Routing uses `useRoutes` inside `BrowserRouter` (app) and `MemoryRouter` (tests) instead of `createBrowserRouter`: the data-router `Request` path fails under jsdom (`Expected signal to be an instance of AbortSignal`). `routes` is exported for reuse; `routerFuture` opts into the v7 flags.
- Token is stored in localStorage (`energy.token`), as tasks 6.9 specify, plus `energy.username` for the topbar; a 401 (except on login) clears both and signs out through `AuthProvider`.
- `AppShell` renders `NavLink`s (navigation markup) and stays hook-free; the nav list comes from `app/navigation.tsx` and only has Dashboard until slices 7 and 8 add their routes (no dead links).
- Extra atoms beyond the task list: `Icon` (inline SVG set), `EmptyState`, `ErrorState`. `lib/tone.ts` holds the static Tailwind class maps per token.
- Temporary run trigger (replaced by `useAnalysisJob` in slice 8): `DashboardPage` starts the analysis, refetches the dashboard once, and polls it every second for up to 30 s until a newer completed analysis appears; the trigger is not disabled while waiting (the server is single-flight, a second click returns the same id).
- Design token `--muted` darkened from `#64748B` to `#5B6B80` for AA contrast; teal `#0EA5A4` is reserved for non-text use with `#0F766E` for buttons/links (contrast). Recorded above.
- Size: 62 files changed, 2,447 additions (lockfile excluded), about 900 of them tests, in 11 commits. `size:exception` accepted for all PRs.

## Batch 8: Slice 7 (complete)

Completed tasks: 7.1 - 7.10 (104/125 overall).

### Slice 7 (branch `feat/07-meters-detail`, base `feat/06-web-shell-dashboard`)

Commits (work units):
| Hash | Message |
|------|---------|
| cedbbae | build(frontend): add recharts 3.10.1 for meter charts |
| 8abf928 | feat(frontend): add meters API client and query hooks (7.1, 7.2) |
| 0ef35c0 | feat(frontend): add meters list with URL-driven search, status filter and sort (7.3, 7.4, 7.5) |
| 0f61cbd | feat(frontend): add chart data helpers for baseline series, windows, events and zoom (7.6) |
| 7f54a8b | feat(frontend): add MetricChart with baseline overlay, tooltip and text alternative (7.7) |
| 41ce69c | feat(frontend): add ChartGrid with shared time range and EventTimeline (7.8) |
| 5a84cc9 | refactor(frontend): extract the run-analysis trigger into a shared hook (7.9) |
| ca4d262 | feat(frontend): add meter detail page with charts, events and routes (7.9) |
| 2db2a43 | style(frontend): widen chart y-axis so unit labels do not overlap ticks |

### TDD Cycle Evidence (slice 7)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 7.1/7.2 | `api/meters.test.ts` | `Failed to resolve import "./meters"` | 5 passed | no filters, all four, empty skipped, items unwrapped, encoded id for detail/readings/events | none |
| 7.3-7.5 | `SearchFilterBar.test.tsx`, `MetersTable.test.tsx`, `MetersView.test.tsx`, `MetersPage.test.tsx`, `status.test.ts` (variationTone) | 4 files `Failed to resolve import`, `variationTone is not defined` | 93 passed | typing `109` leaves M-109 and sets `?q=109`; shared link with status+sort+order; status and sort through the URL; clear filters; error then Retry; null variation placeholder; row links | MemoryRouter tests adopt router future flags (no warnings) |
| 7.6/7.7 | `lib/chartData.test.ts`, `MetricChart.test.tsx`, `format.test.ts` (axis tick, number) | `Failed to resolve import "./chartData"` / `"./MetricChart"`; formatAxisTick not defined | 15 + 17 passed | per-metric baseline pairing, 0 or missing baseline is null, delta, areas by severity, events sorted and labelled, zoom margin 24 h and custom, clip inclusive, summary and empty; tooltip with/without baseline and empty | none |
| 7.8 | `ChartGrid.test.tsx`, `EventTimeline.test.tsx`, `chartData.test.ts` (latestDayTotals) | `Failed to resolve import "./ChartGrid"` / `"./EventTimeline"`, 3 latestDayTotals failures | 67 passed | four charts by name, empty on all four, chronological events, empty list | none |
| 7.9 | `MeterDetailView.test.tsx`, `MeterDetailPage.test.tsx`, `router.test.tsx` (meters routes) | `Failed to resolve import "./MeterDetailPage"`, 2 route tests failed | 141 passed | KPIs, 4 charts, loading, not-found with link, error+Retry, section error keeps meter, parallel fetch (3 requests), run refreshes meter after new analysis, nav current on list and detail | `useRunAnalysis` extracted from DashboardPage (its 8 tests unchanged and green) |
| 7.10 verify | n/a | n/a | tsc clean, 141 tests, build ok, go tests ok | n/a | done |

### Work Unit Evidence (slice 7)
| Evidence | Value |
|---|---|
| Focused test command | `cd frontend && npx vitest run --reporter=dot`: 22 files, 141 tests passed; `npx tsc --noEmit` clean; `npm run build` ok; `cd backend && go test ./... -count=1`: 9 packages ok |
| Runtime harness | Real API binary (analysis run first, so M-109 is Critical) + Vite with `API_TARGET`; headless Chrome script in the scratchpad (not committed): UI login to `/`, Meters nav to `/meters` (12 rows), typing `109` gives `?q=109` and only M-109, status Critical gives `?status=Critical` with only M-109, `?sort=variation&order=desc` puts M-109 (+110.7%), M-104 (+47.4%), M-103 first, row click opens `/meters/M-109` with 4 charts (each with a dashed baseline line and text alternative), rise visible from 12 Sep, event marker "Unknown" at 2026-09-12 14:00 UTC, KPIs 17,526.0 kWh / +110.7% / 2,207.6 kWh vs baseline ~1,047.7 kWh / 1 finding; 390px: no horizontal overflow, charts stacked; `/meters/M-999` shows "Meter not found". Servers stopped. |
| Rollback boundary | `frontend/src/{api/meters.ts,hooks/{useMeters,useMeterReadings,useRunAnalysis}.ts,lib/{chartData,chartTheme}.ts,components/{molecules/SearchFilterBar,organisms/{MetersTable,MetricChart,ChartGrid,EventTimeline}}.tsx,pages/{Meters,MeterDetail}/,app/{router,navigation}.tsx}`, recharts in `package.json` |

### Deviations / notes (slice 7)
- No anomaly windows yet: `MetricChart`/`ChartGrid` accept `areas`/`windows` (severity-tinted `ReferenceArea`, tested at the data-mapping level); slice 8 wires them to anomalies. Event markers are drawn on the consumption chart only.
- Meter detail shows a KPI "Open findings" (API `anomaly_count`) rather than a link, because `/anomalies` does not exist until slice 8 (no dead links). Slice 8 task 8.9 adds the section.
- Header numbers come from the API (`total_kwh`, `variation_pct`) plus "Latest 24 h" and "Baseline ~x kWh" derived from the last 24 readings (`latestDayTotals`), since the API has no daily baseline field.
- Sort: URL uses `sort` and `order` (defaults omitted); the select combines them (`variation:desc`). Status sort offers only "most severe first".
- `useRunAnalysis` replaces the inline slice 6 logic: it now reads the last analysis id via `fetchQuery` before POST and, when a newer analysis lands, invalidates `meters` and `meter`. The dashboard is only observed while a run is watched on the meter page.
- `useDashboard` gained an `enabled` argument. Recharts 3.10.1 pinned; animations are disabled (respects reduced motion by construction). Chart colors come from tokens through `lib/chartTheme.ts` (`rgb(var(--token))`).
- Tests mock only `ResponsiveContainer` (fixed size) so the real chart renders in jsdom; assertions use accessible names and text, not SVG geometry.
- Size: see return summary; `size:exception` accepted for all PRs.

## Batch 9: Slice 8 (complete)

Completed tasks: 8.1 - 8.10 (114/125 overall), plus the approved y-axis polish.

### Slice 8 (branch `feat/08-analysis-anomalies-investigation`, base `feat/07-meters-detail`)

Commits (work units):
| Hash | Message |
|------|---------|
| 290cfc4 | chore(frontend): remove stray sed backup file |
| 4bf9c09 | feat(frontend): add analysis API types and useAnalysisJob with 500 ms polling (8.1, 8.2) |
| 656a92c | feat(frontend): add StageStepper and AnalysisProgressPanel with live announcements (8.3) |
| 7ce9777 | feat(frontend): wire live analysis progress into dashboard and meter detail (8.4) |
| fd1d68f | feat(frontend): add anomalies API client, hooks and window and action formatters (8.5) |
| 37dca8f | feat(frontend): add ranked AI Anomalies page with URL filters, empty states and route (8.6) |
| 8d8d319 | feat(frontend): let charts zoom to a range and describe highlighted anomaly windows in text (8.8) |
| 6ae8441 | feat(frontend): add investigation page with evidence table, related event and recommended action card (8.7, 8.8) |
| 1e7a5e3 | feat(frontend): show meter anomalies, highlight their windows and link open findings (8.9) |
| a07afb6 | fix(frontend): use per-metric y-axis precision with whole-number amps |

### TDD Cycle Evidence (slice 8)
| Task | Test file | RED (observed) | GREEN | TRIANGULATE | REFACTOR |
|------|-----------|----------------|-------|-------------|----------|
| 8.1/8.2 | `hooks/useAnalysisJob.test.tsx` (fake timers) | `Failed to resolve import "./useAnalysisJob"` | 9 passed on the first run | refetch predicate (5 statuses), pending to running to done with 500 ms ticks and no polling after done, invalidation of 5 query keys, no second POST while active, resume from `/latest` when running, finished latest ignored, failed then restart, start error | none |
| 8.3 | `AnalysisProgressPanel.test.tsx` | `Failed to resolve import "../molecules/StageStepper"` | 12 passed | stepper states done/current/pending/failed, pending run, stage 3 vs 6 announcement, headline never hardcoded, failed with and without reason | none |
| 8.4 | `DashboardPage.test.tsx`, `MeterDetailPage.test.tsx`, `MeterDetailView.test.tsx` | 8 failures (no panel, no "Analyzing" label, `busy` prop) | 168 passed | progress through stages to headline and link, disabled trigger with progress, in-flight request, start error, failed then Retry, resume on mount (both pages), meter refresh after done | `useRunAnalysis` and the dashboard watch logic deleted; `RunAnalysisButton`/`RunAnalysisFeedback` shared; `act` warnings fixed by using RTL `waitFor` |
| 8.5/8.6 | `api/anomalies.test.ts`, `AnomaliesTable.test.tsx`, `AnomaliesView.test.tsx`, `AnomaliesPage.test.tsx`, `format`/`status` (window, action label), `router.test.tsx` | unresolved imports, 5 failing helper tests, route test could not find the nav link | 199 passed | four ranked rows, URL filters incl. `meter_id`, filter then clear, no-analysis empty state then run then table, no-match vs nothing-detected, loading, error plus Retry | none. Note: `AnomaliesPage.test` passed on its first run because the page was written with the view; RED evidence for that unit is the route test |
| 8.7/8.8 | `EvidenceTable`, `ActionCard`, `InvestigationView`, `InvestigationPage` tests, `MetricChart`/`ChartGrid`/`chartData` tests | unresolved imports, window label and range tests failing | 235 passed | M-109 (no explaining event), M-106 (scheduled outage card), null event, 106-reading zoom, loading/not-found/error, readings-only failure, readings not requested before the meter is known | `SectionState`/`toSection` moved to `lib/section.ts`; `variationClass` moved to `lib/tone.ts` |
| 8.9 | `MeterDetailView.test.tsx`, `MeterDetailPage.test.tsx` | 9 failures | 244 passed | findings link present/absent, section rows, empty, windows in chart text, section error, loading, anomalies fetched in parallel | none |
| polish | `format.test.ts`, `chartData.test.ts` | `formatAxisNumber is not a function`, `axis` undefined | 250 passed | A 525/524.6, PF 0.90/1.00, V 230/229.5, kWh 90/0.5 | DOM tick tests dropped: Recharts renders no tick text in jsdom, so the pure formatter is tested instead |

### Work Unit Evidence (slice 8)
| Evidence | Value |
|---|---|
| Focused test command | `cd frontend && npx vitest run --reporter=dot`: 32 files, 250 tests passed (141 before the slice); `npx tsc --noEmit` clean; `npm run build` ok; `cd backend && go test ./... -count=1`: 9 packages ok (unchanged) |
| Runtime harness | Real API (fresh temp DB, `ANALYSIS_STAGE_DELAY_MS=300`, port 18085) + Vite (`API_TARGET`), headless Chrome CDP script in the scratchpad (not committed): login to `/`, Meters (12 rows), M-109, Run AI Analysis; mid-run "Stage 3 of 7: Detection", bar 28, stepper (2 done, Detection in progress, 4 pending), button "Analyzing... 28%" disabled; done headline `4 anomalies detected · 2 require priority attention` from the API with a "View anomalies" link; M-109 detail then shows 1 open finding with a link, an "Anomalies for this meter" table and a highlighted window on all four charts; `/anomalies` 4 rows in order M-109, M-112, M-104, M-106; M-109 investigation: REAL_ANOMALY/High/98%, 8 evidence rows, "No operational event explains this change", action card `INSPECT_LOAD_INCREASE`, Priority #1, charts zoomed to 82 readings with the window highlighted; M-106 shows the Scheduled outage event; `?type=DATA_QUALITY` leaves only M-112. Servers stopped. |
| Rollback boundary | `frontend/src/{api/{analysis,anomalies}.ts,hooks/{useAnalysisJob,useAnomalies}.ts,components/{molecules/{StageStepper,RunAnalysisButton,EvidenceRow},organisms/{AnalysisProgressPanel,RunAnalysisFeedback,AnomaliesTable,EvidenceTable,ActionCard,RelatedEventCard}},pages/{Anomalies,Investigation}/,lib/section.ts}` plus edits to Dashboard/MeterDetail pages, router, navigation, chart files |

Screenshots (scratchpad, not committed): `progress.png`, `anomalies.png`, `investigation.png`, `m109-windows.png`.

### Deviations / notes (slice 8)
- Hook test file is `useAnalysisJob.test.tsx` (JSX wrapper), not `.ts`.
- Nav label is "AI Anomalies" (matches page title and demo flow).
- Anomaly row action column shows a short label from the action code (`actionLabel`); the full text lives on the ActionCard. Table cell uses a stretched "Investigate" link with `aria-label="Investigate <meter>"`.
- `useAnalysisJob` resumes via `GET /api/ai/analysis/latest` (query with `gcTime: 0`, so a remount never trusts a stale "running"); the trigger guard is `active`, and the dashboard summary watch logic of slices 6 and 7 was removed (done invalidates `dashboard`, `meters`, `meter`, `anomalies`, `anomaly`).
- The finished run's panel carries the headline, so the dashboard and anomalies banners are hidden while it is shown (no duplicate text).
- UNKNOWN related events (M-109) are shown as "No operational event explains this change" and are not drawn as a chart marker; the `event` evidence row is left to the related-event card.
- Charts: `ChartGrid` accepts `range` (zoom, window +-24 h) and windows with `label`; windows are also spelled out in each chart's text alternative and legend ("Anomaly window"), so severity tint is not the only signal.
- Polish: y-axis precision per metric (kWh 0-1 dp, V 0-1 dp, A integers with `allowDecimals={false}`, PF 2 dp).
- Removed the tracked stray file `frontend/src/app/router.test.tsx-E` (sed backup from slice 7).
- Backend gotcha: a stale API process from an earlier session can keep port 18085, so a harness run silently hits the old DB; check `lsof -i :18085` first.
- Size: 56 files, 2,733 additions and 311 deletions (lockfile excluded), about half tests, in 10 commits. `size:exception` accepted for all PRs.

## Batch 9: Slice 9 Delivery (complete) plus explainer fix

Completed tasks: 9.1 - 9.11 (125/125 overall). Branch `feat/09-delivery` (base `feat/08-analysis-anomalies-investigation`).

| Hash | Message |
|------|---------|
| 104f2cc | fix(explain): render power factor and ratio with two decimals |
| b5f2166 | feat(delivery): add Dockerfiles, nginx config and zero-config compose stack |
| 72be469 | docs: add README with architecture, threshold derivation and validation results |
| (this commit) | docs(openspec): mark slice 9 complete and record apply progress |

### TDD Cycle Evidence (explainer fix)
| Task | Test file | RED (observed) | GREEN | REFACTOR |
|------|-----------|----------------|-------|----------|
| fix | `backend/internal/adapters/explain/template_test.go` `TestPowerFactorRendersWithTwoDecimals` | reason and action lacked "0.94" and "0.74" (rendered "0.9" and "0.7") | `ok energy/internal/adapters/explain`; full backend suite 9 packages ok | new `ratio()` helper (2 decimals) used for power factor and hourly-pattern correlation; kWh, V, A stay at 1 decimal. The electrical-ratio detail was already 2 decimals (`score.go`). |

### Work Unit Evidence (slice 9)
| Evidence | Value |
|---|---|
| Focused test command | `cd backend && go vet ./... && gofmt -l . && go test ./... -count=1`: 9 packages ok; `cd frontend && npx tsc --noEmit` clean; `npx vitest run --reporter=dot`: 32 files, 250 tests passed |
| Runtime harness | `docker compose config -q` ok; `docker compose up --build -d` (api healthy before web starts); via `http://localhost:8080`: `/` 200 text/html, `/meters/M-109` returns index.html, `/api/health` `{"status":"ok"}`, hashed asset served gzip with `Cache-Control: public, immutable`; login ok; 12 meters; `POST /api/ai/analyze` then polling through readings, baseline, correlation, explanation, recommendation to done; `GET /api/anomalies` returns 4 items (M-109 REAL HIGH, M-112 DQ HIGH, M-104 EXPLAINABLE MEDIUM, M-106 FP LOW) with summary `4 anomalies detected · 2 require priority attention`; M-109 reason now says "power factor moved from 0.94 to 0.74". `down` then `up` kept the 4 anomalies; `down -v` then `up` reseeded (12 meters, no analysis). Images: web 50.6 MB, api 19.7 MB. Stack removed with `down -v`. |
| Rollback boundary | `backend/Dockerfile`, `backend/.dockerignore`, `frontend/Dockerfile`, `.dockerignore`, `deploy/nginx.conf`, `docker-compose.yml`, `.env.example`, `README.md`; `docs/thresholds.md` deletion (content folded into README); explainer change in `backend/internal/adapters/explain/template.go` and its test |

### Deviations / notes (slice 9)
- `backend/Dockerfile` uses `golang:1.23.4-alpine` with `GOTOOLCHAIN=local` (go.mod pins toolchain go1.23.4), `-ldflags="-s -w"`, and `alpine:3.20` runtime with non-root user `app`. CSVs are not baked in: compose mounts `./data:/data:ro`.
- `frontend/Dockerfile` build context is the repository root (compose `context: .`, `dockerfile: frontend/Dockerfile`) so `deploy/nginx.conf` is reachable; it uses `nginx:1.27-alpine`. A root `.dockerignore` (excludes backend, docs, openspec, node_modules, dist) serves that build; `backend/.dockerignore` serves the backend build.
- Compose adds an optional `WEB_PORT` (default 8080) and `${VAR:-default}` interpolation for demo credentials, secret, TTL and stage delay, so `.env` is optional. `ANALYSIS_STAGE_DELAY_MS` defaults to 300 in compose.
- nginx: gzip, `/assets/` cached one year immutable, `index.html` no-cache, history fallback, `/api/` proxied.
- 9.9: `rg -n "expected[_]results" . --glob '!node_modules' --glob '!.git'` still matches two SDD artifacts under `openspec/` (proposal out-of-scope bullet and the platform-delivery spec scenario that describes this very check). Task 9.9 excludes `openspec/**`; with that glob it returns nothing. Build, tests and runtime read only `data/readings.csv` and `data/events.csv`.
- 9.7: `docs/thresholds.md` removed; no remaining links to it outside `openspec/`.
- The task 9.11 checklist in `proposal.md` was left untouched (optional).

## Remaining
None. All 125 tasks complete. Next: archive or optional verify; PR creation is the user's call.
