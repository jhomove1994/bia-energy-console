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

## Remaining
Slices 4a - 9 (73 tasks) not started.
