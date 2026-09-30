# Tasks: AI Energy Management Platform (MVP)

Change: `ai-energy-platform` | Store: hybrid | Base branch: `main` | Design: `design.md` (authoritative, incl. D4)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~4,500 authored (tests and docs included; CSV copies, lockfiles excluded) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Slice 0 (baseline commit) then PR 1 -> PR 2 -> PR 3 -> PR 4a -> PR 4b -> PR 5 -> ... -> PR 9 |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No (resolved 2026-09-30: split slice 4 into 4a/4b; `size:exception` accepted for ALL PRs, see below)
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

Delivery decision (user, 2026-09-30): actual sizes run about 3.5x the forecasts (PR 1 = 1,463 authored lines; PR 2 = 2,854 added lines of which 1,459 are tests). The user ACCEPTED `size:exception` for ALL PRs. Review happens commit by commit, so every commit stays a small, cohesive work unit (behavior + tests together). The 4a/4b split of slice 4 stays.

Do not shrink code, tests, or docs to fit the budget. Report overage instead.

### Delivery mechanics (stacked-to-main)

- Slice N branches from the slice N-1 branch until N-1 merges; after merge, retarget slice N to `main` and rebase so the diff shows only its own work unit.
- Every PR body carries a dependency diagram with a marker on the current PR, plus start, end, prior dependencies, follow-up, out-of-scope.
- Commits are work units (behavior + tests + docs together), Conventional Commits, no AI attribution.
- Strict TDD: every engine, app, adapter, and API task is RED (failing test observed) then GREEN then REFACTOR. Runner: `go test ./... -count=1` (summary output only, never `-v`). Frontend runner (from slice 6): `npx vitest run --reporter=dot`.
- HARD RULES: never read or copy-inspect `docs/readings.csv` or `docs/events.csv` content by hand; never reference the expected-results file in code, config, tests, or docs.

### Spec-vs-design naming resolutions (spec wins for public API field names)

- List query param is `q` (spec); also accept `search` as alias. Sort/order per design (`sort=meter_id|variation|kwh|status`, `order=asc|desc`).
- Dashboard fields: `meters_count`, `total_kwh`, `anomalies_count`, `high_priority_count`, `aggregate_confidence` (nullable), `last_analysis` (nullable), `summary_text` (nullable).
- Readings point fields: `timestamp`, `consumption_kwh`, `voltage_v`, `current_a`, `power_factor`, `status`, `baseline_kwh` (required) plus additive `baseline_voltage_v`, `baseline_current_a`, `baseline_power_factor`.
- Anomaly rank field: `priority_rank`. Meter list includes `total_kwh` and `variation_pct`.
- Analyze response: `{ "id": "<uuid>" }`.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 0 | `main` baseline commit | none (direct branch setup) | `git log --oneline main` | `git status --short` clean | n/a (first commit) |
| 1 | Backend foundation: config, domain, SQLite, CSV seed, wiring | PR 1 (`feat/01-backend-foundation`, base `main`) | `cd backend && go test ./... -count=1` | `DATA_DIR=../data DB_PATH=$(mktemp -d)/e.db go run ./cmd/api` logs `seed: meters=12 readings=4032 events=4` | `backend/`, `data/` |
| 2 | Baseline + read API + auth + dashboard (pre-analysis) | PR 2 (`feat/02-read-api-auth`, base PR 1) | `cd backend && go test ./... -count=1` | login then `GET /api/meters` with bearer via curl against `go run ./cmd/api` | `backend/internal/{app,adapters/http,engine/baseline.go,engine/config.go}` + sqlite read repos |
| 3 | Engine detectors | PR 3 (`feat/03-engine-detectors`, base PR 2) | `cd backend && go test ./internal/engine/... -count=1` | N/A: pure library, proven by synthetic tests | `backend/internal/engine/{detect,quality,pattern}*.go` |
| 4a | Engine correlate, events, classify | PR 4a (`feat/04a-engine-correlate-classify`, base PR 3) | `cd backend && go test ./internal/engine/... -count=1` | N/A: pure library, proven by synthetic tests | `backend/internal/engine/{correlate,events,classify}*.go` |
| 4b | Engine score, recommend, explainer, Analyze, dataset test | PR 4b (`feat/04b-engine-score-dataset`, base PR 4a) | `cd backend && go test ./internal/engine/... ./internal/adapters/explain/... -count=1` | dataset test loads `data/*.csv` end to end (4 anomalies) | `backend/internal/engine/{score,recommend,engine,guard}*.go`, `adapters/explain/`, `docs/thresholds.md` |
| 5 | Analysis jobs + anomalies API + status derivation | PR 5 (`feat/05-analysis-jobs-api`, base PR 4b) | `cd backend && go test ./... -count=1` | curl: login, `POST /api/ai/analyze`, poll to done, `GET /api/anomalies` returns 4 | `backend/internal/{app/analysis_service.go,app/summary.go,adapters/http/handlers_analysis.go,handlers_anomalies.go,adapters/sqlite/{analyses,anomalies}.go}` |
| 6 | Web shell + login + dashboard | PR 6 (`feat/06-web-shell-dashboard`, base PR 5) | `cd frontend && npx vitest run --reporter=dot` | `npm run build`; Vite dev + API: login then dashboard renders KPIs | `frontend/` |
| 7 | Meters list + meter detail charts | PR 7 (`feat/07-meters-detail`, base PR 6) | `cd frontend && npx vitest run --reporter=dot` | Vite dev + API: filter/sort meters, open M-109, 4 charts render | `frontend/src/{pages/Meters*,pages/MeterDetail*,components/organisms/{MetersTable,MetricChart,ChartGrid,EventTimeline}}` |
| 8 | Run analysis progress, anomalies, investigation | PR 8 (`feat/08-analysis-anomalies-investigation`, base PR 7) | `cd frontend && npx vitest run --reporter=dot` | Vite dev + API: run analysis to done, open M-109 investigation | `frontend/src/{hooks/useAnalysisJob.ts,pages/Anomalies*,pages/Investigation*,organisms}` |
| 9 | Docker, nginx, compose, README with thresholds | PR 9 (`feat/09-delivery`, base PR 8) | `cd backend && go test ./... -count=1` | `docker compose up --build`, browse `http://localhost:8080`, full demo flow | `backend/Dockerfile`, `frontend/Dockerfile`, `deploy/`, `docker-compose.yml`, `README.md` |

---

## Slice 0: Baseline on `main` (not a PR)

Branch: `main`. Goal: create `main` with the baseline commit. Forecast: 0 authored lines (docs and specs only).
Covers: proposal Resolved Decision 4.

- [x] 0.1 Rename branch `master` to `main` (`git branch -m master main`); confirm `.gitignore` covers `backend/*.db`, `backend/energy.db`, `node_modules/`, `frontend/dist/`, `*.log` by editing `.gitignore` if missing.
- [x] 0.2 Stage `.gitignore`, `docs/`, `openspec/` and create the baseline commit `chore: add baseline docs, specs and gitignore` on `main`. Verify: `git log --oneline main` shows one commit and `git status --short` is empty.

Acceptance: `main` exists with one commit; working tree clean.

---

## Slice 1: Backend foundation

Branch: `feat/01-backend-foundation` (base `main`). Forecast: ~420 lines. Task count: 16.
Spec coverage: data-ingestion (all requirements), platform-delivery (Documented configuration, Source-data boundaries), auth-session (Zero-config credentials: config defaults only).

- [x] 1.1 Create `backend/go.mod` (`module energy`, `go 1.23`, `toolchain go1.23.x`), add `modernc.org/sqlite` pinned to the newest `v1.34.x` whose `go` directive is <= 1.23, else `v1.33.x`. Verify with `cd backend && GOTOOLCHAIN=auto go mod tidy && go build ./...` (records the resolved version in the commit message).
- [x] 1.2 Copy the source CSVs unchanged: `cp docs/readings.csv data/readings.csv` and `cp docs/events.csv data/events.csv`, creating `data/readings.csv` and `data/events.csv` (sources `docs/readings.csv` (read-only), `docs/events.csv` (read-only)). Do not open or print their content.
- [x] 1.3 RED: `backend/internal/platform/config_test.go` asserts defaults `PORT=8080 DB_PATH=./energy.db DATA_DIR=../data SEED_ON_START=true DEMO_USER=admin DEMO_PASSWORD=admin123 AUTH_SECRET=<dev constant> TOKEN_TTL=12h ANALYSIS_STAGE_DELAY_MS=0` (compose overrides to 300), env overrides, and invalid duration/int returns an error. Test fails (no package).
- [x] 1.4 GREEN: `backend/internal/platform/config.go` with `Load(getenv func(string) string) (Config, error)`; `Config.String()` redacts password and secret.
- [x] 1.5 RED: `backend/internal/domain/event_test.go` for `NormalizeEventType`: `FIRMWARE_UPDATE` -> `UNKNOWN`, `maintenance`/`SCHEDULED_MAINTENANCE`/`PLANNED_OUTAGE` -> `SCHEDULED_OUTAGE`, trims and uppercases, known types unchanged.
- [x] 1.6 GREEN: create `backend/internal/domain/{meter,reading,event,anomaly,evidence,analysis}.go` per design section 1.2 (types and enums, no dependencies; Meter has only `MeterID` and `CreatedAt`), with `NormalizeEventType`.
- [x] 1.7 RED: `backend/internal/adapters/sqlite/migrate_test.go` (temp-dir DB): after `Open+Migrate`, `PRAGMA user_version=1`, all 5 tables exist, `foreign_keys=1`, `journal_mode=wal`, second `Migrate` is a no-op.
- [x] 1.8 GREEN: `backend/internal/adapters/sqlite/{db.go,schema.sql,migrate.go}` (embedded schema from design section 3, `SetMaxOpenConns(1)`, DSN pragmas WAL, busy_timeout 5000, foreign_keys 1).
- [x] 1.9 RED: `backend/internal/adapters/csvload/csvload_test.go` with inline fixtures: valid rows in formats `2006-01-02 15:04:05`, `2006-01-02 15:04`, RFC3339 (timezone-less = UTC); a row with non-numeric `voltage_v` is skipped and logged with file, line, reason; wrong column count skipped; other rows still parsed; unrecognized event type stored as `UNKNOWN` with original description.
- [x] 1.10 GREEN: `backend/internal/adapters/csvload/{readings.go,events.go}` (parsers returning rows plus a skipped-row counter, logger injected).
- [x] 1.11 RED: `backend/internal/adapters/csvload/seed_test.go` with synthetic temp CSVs: seed twice -> identical counts (idempotent); partially seeded DB gets only missing rows, existing rows unchanged; missing `readings.csv` returns an error naming the file; missing timestamp (05:00) creates no row and the log reports 1 gap for that meter; status `SENSOR_FAULT` stored as given; meter table has only `meter_id` and `created_at` columns; no readings interpolated.
- [x] 1.12 GREEN: `backend/internal/adapters/csvload/seed.go` (one tx, `INSERT OR IGNORE` meters then readings then events, per-meter gap log, log line `seed: meters=N readings=N events=N (inserted=N)`) and the write methods in `backend/internal/adapters/sqlite/{meters,readings,events}.go` used by seed.
- [x] 1.13 RED then GREEN: `backend/internal/adapters/csvload/dataset_seed_test.go` seeds `data/*.csv` into a temp DB and asserts 12 meters, 4,032 readings, 4 events, zero gaps, zero non-OK statuses (counts via SQL only).
- [x] 1.14 Wire `backend/cmd/api/main.go`: config -> open sqlite -> migrate -> seed when `SEED_ON_START` (exit non-zero if DB empty and CSV missing; warn and continue if DB already holds data) -> log config summary without secrets. No HTTP yet.
- [x] 1.15 REFACTOR: `gofmt -l backend` empty and `go vet ./...` clean; tests still green.
- [x] 1.16 Re-run `sdd-init` now that `go test` exists so the cached `sdd-init/prueba-tecnica` reports `strict_tdd: true` with runner `go test ./... -count=1`; update `openspec/config.yaml` `strict_tdd: true` if sdd-init does not. Verify by reading the new `sdd-init/prueba-tecnica` observation.

Verification: `cd backend && go test ./... -count=1` (all packages ok); `rg -n "expected[_]results" backend data` returns nothing; run harness from the work-unit table (seed log shows 12/4032/4; second run inserted=0).

---

## Slice 2: Baseline, read API, auth, dashboard (pre-analysis)

Branch: `feat/02-read-api-auth` (base slice 1 branch). Forecast: ~560 lines (size:exception risk). Task count: 17.
Spec coverage: meter-catalog (all, status always OK pre-analysis), auth-session (all), dashboard-summary (Before any analysis), anomaly-engine (Baseline per meter and hour of day, first half of Configurable parameters), platform-delivery (Automated verification for login/meters/detail/readings/dashboard endpoints).
Design deviation D4: `engine/baseline.go` lives here.

- [x] 2.1 RED: `backend/internal/engine/baseline_test.go`: hour-of-day median/MAD over the first 7 days; MAD floor (`max(MAD, 3% median)`); `DailyKWh` = sum of hourly medians; `RatioMedian`; `Profile[24]`; meter with < 5 samples per hour is `insufficient_history` (skipped, not an anomaly); series shifted by 30 days gives identical baseline; no `time.Now` used.
- [x] 2.2 GREEN: `backend/internal/engine/config.go` (Config struct with baseline fields `BaselineDays=7 BaselineMinSamples=5 MADScale=1.4826 MADFloorPct=0.03` and `DefaultConfig()`) and `backend/internal/engine/baseline.go` (`BuildBaseline`), imports `domain` and stdlib only.
- [x] 2.3 Create `backend/internal/app/ports.go` with `MeterRepository`, `ReadingRepository`, `EventRepository`, `Clock`, and `MeterStatusSource` (returns per-meter status and anomaly counts; slice 5 supplies the real one; here a `NoAnalysis` default returns all OK).
- [x] 2.4 RED: `backend/internal/adapters/sqlite/read_repos_test.go`: `List`, `Exists`, `ByMeter` (ascending, `from`/`to` inclusive range), `AllByMeter` sorted by ts, `TotalKWh` equals the sum, events by meter, using a synthetic seeded temp DB.
- [x] 2.5 GREEN: read methods in `backend/internal/adapters/sqlite/{meters,readings,events}.go`.
- [x] 2.6 RED: `backend/internal/adapters/http/auth_test.go`: token round trip; tampered payload rejected; expired rejected; token issued before "restart" (new verifier, same secret) still valid; different secret rejected; compare uses constant time (function under test is `Verify`).
- [x] 2.7 GREEN: `backend/internal/adapters/http/auth.go` (`base64url(json{sub,exp}).base64url(HMAC-SHA256)`, `crypto/subtle`).
- [x] 2.8 RED: `backend/internal/adapters/http/auth_http_test.go` (httptest): login 200 `{token, expires_at}`, wrong password 401 with no token, malformed JSON 400, protected route without/malformed/tampered/expired token 401, valid token 200, `GET /api/health` public 200, panic in handler returns 500 JSON, startup log lines contain neither password nor secret.
- [x] 2.9 GREEN: `backend/internal/adapters/http/{router.go,middleware.go,errors.go,handlers_auth.go,handlers_health.go}` (stdlib `ServeMux` method patterns; chain `recover -> requestLog -> auth`; error shape `{"error":{"code","message"}}`).
- [x] 2.10 RED: `backend/internal/app/meter_service_test.go` with fake repos: list rows have `meter_id`, `last_reading_at`, `total_kwh`, `variation_pct`, `status`; variation = latest full day vs baseline `DailyKWh` (about +109% on a synthetic tripled day, within +-5% on a healthy one); status OK for all with `NoAnalysis`; `q` case-insensitive substring; invalid `status`/`sort` returns a typed validation error; detail includes findings count; unknown id returns `ErrNotFound`; readings carry `baseline_kwh`, range filter, inverted range error.
- [x] 2.11 GREEN: `backend/internal/app/meter_service.go` (uses `engine.BuildBaseline`).
- [x] 2.12 RED: `backend/internal/adapters/http/meters_http_test.go` (real sqlite temp DB seeded from a synthetic fixture generator in `backend/internal/adapters/http/testutil_test.go`): `GET /api/meters` 200 with 12 meters no name/location; `q=m-10` filter; `status=BROKEN` 400; `sort` and `order`; `GET /api/meters/{id}` 200 and 404; `/readings` full series ascending with `baseline_kwh`, day range returns exactly 24, unknown meter 404, inverted range 400, unparsable range 400; `/events` 200 and 404.
- [x] 2.13 GREEN: `backend/internal/adapters/http/{handlers_meters.go,dto.go}` (DTOs per naming resolutions above).
- [x] 2.14 RED: `backend/internal/app/dashboard_service_test.go` and endpoint test: `GET /api/dashboard/summary` before any analysis returns `meters_count`, `total_kwh` equal to the sum of all readings, `anomalies_count=0`, `high_priority_count=0`, and `aggregate_confidence`, `last_analysis`, `summary_text` all null.
- [x] 2.15 GREEN: `backend/internal/app/dashboard_service.go` (with an injectable `LatestDone` source defaulting to none) and `backend/internal/adapters/http/handlers_dashboard.go`.
- [x] 2.16 Wire HTTP server in `backend/cmd/api/main.go` (start listening only after seed completes; graceful shutdown).
- [x] 2.17 REFACTOR + verify: `gofmt -l backend`, `go vet ./...`, `go test ./... -count=1`; `rg -n "M-\d+" backend/internal/engine --glob '!*_test.go'` returns nothing.

---

## Slice 3: Engine detectors

Branch: `feat/03-engine-detectors` (base slice 2 branch). Forecast: ~520 lines (size:exception risk). Task count: 17.
Spec coverage: anomaly-engine (Tiny MAD does not fire, Consumption effect gate, Persistence and spikes, Daily variation evidence, Data-quality oscillation detection, Gap and status detection, Electrical consistency ratio, Hourly pattern break as evidence (detector part), Configurable parameters).

- [x] 3.1 RED: `backend/internal/engine/config_test.go` asserts every `DefaultConfig()` value from design section 2.2 and that overriding `HourlyDevPct` to 0.60 changes results of a small synthetic series.
- [x] 3.2 GREEN: complete `backend/internal/engine/config.go` with all thresholds from design section 2.2 (short names only; per-meter justifications stay in README, no meter ids in comments).
- [x] 3.3 Create test helper `backend/internal/engine/synth_test.go`: `synth(days, profile, mutate func)` builds hourly readings with V, A, PF, kWh, ratio near 1.05 for synthetic meter ids like `X-1`.
- [x] 3.4 RED: `backend/internal/engine/detect_test.go` hourly tests: +25% with z 8 not flagged; +40% with z 2 not flagged; +47% with z 6 flagged; 12% deviation with near-zero MAD not flagged; 6 consecutive same-sign hours is persistent; 5 hours is a spike; 3 flagged hours is a spike; 4 up then 4 down produces neither persistent; gap of <= 2 unflagged hours merges the run; a single point >= 60% is a spike.
- [x] 3.5 GREEN: `backend/internal/engine/detect.go` (hourly flag, `Signal` type, episode builder, persistent/spike rules).
- [x] 3.6 RED: `detect_test.go` daily variation: day +47% flagged; day +-2% silent; partial day scaled by its own hours' medians.
- [x] 3.7 GREEN: daily variation in `backend/internal/engine/detect.go`.
- [x] 3.8 RED: `backend/internal/engine/quality_test.go` jumps and DQ cluster: V toggling 202/240 V every 3 h for a day is DQ; a single +110% current step is not DQ; three same-direction voltage jumps are not DQ; down then up (one reversal) is not DQ; PF quantized jumps counted; window merge across metrics within 24 h.
- [x] 3.9 GREEN: jumps (V > 8, symmetric current > 50%, PF >= 0.15) and oscillation clustering in `backend/internal/engine/quality.go`.
- [x] 3.10 RED: `quality_test.go` ratio: stable +25% shift is corroboration, not DQ; deviant hours of both signs within 24 h (0.58 to 4.22) is unstable (DQ seed); stable run < 6 h without consumption episode is dropped; ratio skips rows with non-positive denominator.
- [x] 3.11 GREEN: ratio deviation stable/unstable in `backend/internal/engine/quality.go`.
- [x] 3.12 RED: `quality_test.go` gaps and status: a missing hourly timestamp yields a gap signal at that timestamp; `SENSOR_FAULT` yields a status signal; a clean series yields nothing.
- [x] 3.13 GREEN: gap and status runs in `backend/internal/engine/quality.go`.
- [x] 3.14 RED: `backend/internal/engine/pattern_test.go`: a uniform +47% step (correlation about 0.97, shape about 3%) is not a break; a reshaped day (corr < 0.80 and shape > 15%) is a break; partial days skipped.
- [x] 3.15 GREEN: `backend/internal/engine/pattern.go` (Pearson + normalized shape deviation, both required).
- [x] 3.16 RED then GREEN: `backend/internal/engine/purity_test.go` runs `go list -deps` for the engine package and fails if any import is outside `energy/internal/domain` or the stdlib.
- [x] 3.17 REFACTOR + verify: `go vet ./...`, `go test ./... -count=1`; `rg -n "M-\d+" backend/internal/engine --glob '!*_test.go'` returns nothing.

---

## Slice 4: Engine correlate, classify, score, recommend, explainer, dataset test

Split into two PRs (user decision 2026-09-30):
- **PR 4a** `feat/04a-engine-correlate-classify` (base slice 3 branch): tasks 4.1-4.4, closing with `go vet ./...` and `go test ./... -count=1`. Forecast ~330 lines.
- **PR 4b** `feat/04b-engine-score-dataset` (base 4a branch): tasks 4.5-4.15. Forecast ~390 lines.
Task count: 15.
Spec coverage: anomaly-engine (One anomaly per meter per incident window, Event correlation, Ordered classification, Severity, Confidence, Priority ranking, Recommendation, Evidence items, Recovery detection, Healthy meters are silent, Dataset validation outcomes, No hardcoding and determinism, Configurable parameters override), ai-analysis (Template explainer), platform-delivery (Automated verification: engine tests, four cases, healthy silence).

- [ ] 4.1 RED: `backend/internal/engine/correlate_test.go`: hourly flags + daily var + pattern break + current jump + stable ratio on one window give exactly 1 incident; two windows separated by days give 2; secondaries attach but never extend the window; orphan daily variation promotes to a consumption incident; orphan pattern break becomes `PATTERN_CHANGE`; orphan single jumps are dropped; seeds within `AttachSlackHours` merge.
- [ ] 4.2 GREEN: `backend/internal/engine/correlate.go` (`Incident`, seeds/secondaries grouping per design section 2.5).
- [ ] 4.3 RED: `backend/internal/engine/classify_test.go` and `events_test.go`: SCHEDULED_OUTAGE at onset explains a recovered decrease -> `FALSE_POSITIVE`; same outage with a non-recovered decrease -> `EXPLAINABLE_ANOMALY`; outage on an increase does not explain; `UNKNOWN` at onset does not explain (stays REAL, event kept as related context); OPERATIONAL_CHANGE explains persistent shifts of both signs; event 10 h before onset not related; DATA_QUALITY event raises DQ confidence and leaves type unchanged; DQ without any event is `DATA_QUALITY`; DQ takes precedence over a consumption shift; event type normalization; recovery = >= 12 in-gate hours after the window; tie-break by power then closeness.
- [ ] 4.4 GREEN: `backend/internal/engine/{events.go,classify.go}` (match rules, recovery, ordered classification, kind assignment).
- [ ] 4.5 RED: `backend/internal/engine/score_test.go`: bands 70/69/40/39 -> HIGH/MEDIUM/MEDIUM/LOW; EXPLAINABLE raw 85 capped to MEDIUM; FALSE_POSITIVE raw 55 forced LOW; confidence formula input computing 1.2 gives 0.98; confidence label 0.84 is Medium; REAL beats DQ on equal HIGH and confidence; ranking deterministic (ties by severity score then meter id); evidence items carry metric, label, baseline, observed, delta_pct, unit.
- [ ] 4.6 GREEN: `backend/internal/engine/score.go` (severity for consumption and DQ, confidence, priority, `Rank`, evidence builder).
- [ ] 4.7 RED: `backend/internal/engine/recommend_test.go` and `backend/internal/adapters/explain/template_test.go`: every `ActionCode` renders non-empty text differing by type; no `<no value>` in any output; the REAL reason cites observed and baseline values and percent change and states no event explains the change; two renders of the same anomaly are identical; no network or API key needed.
- [ ] 4.8 GREEN: `backend/internal/engine/recommend.go` (`(Type, Sign, Kind) -> ActionCode`) and `backend/internal/adapters/explain/template.go` (`text/template`, numbers to 1 decimal, implements `app.Explainer`; add `Explainer` to `backend/internal/app/ports.go`).
- [ ] 4.9 RED: `backend/internal/engine/engine_test.go`: `Analyze` on synthetic input is deterministic across two runs; stage hook fires in order readings..recommendation subset owned by the engine; overriding `HourlyDevPct` to 0.60 removes findings below 60%.
- [ ] 4.10 GREEN: `backend/internal/engine/engine.go` (`Input`, `Result`, `Analyze(in, cfg, hook)` chaining baseline, detect, correlate, events+classify, score).
- [ ] 4.11 RED: `backend/internal/engine/dataset_test.go` (package `engine_test`, loads `../../../data/*.csv` through `csvload`, test-only): exactly 4 anomalies ranked M-109, M-112, M-104, M-106; M-109 REAL_ANOMALY HIGH rank 1 with UNKNOWN event as context only; M-106 FALSE_POSITIVE LOW with outage as related event; M-112 DATA_QUALITY HIGH, also when its DATA_QUALITY event is removed; M-104 EXPLAINABLE_ANOMALY MEDIUM; the 8 healthy meters yield zero anomalies; dataset with all timestamps shifted 30 days gives identical findings; dataset with every meter id renamed gives identical types, severities, ranks; summary counts 4 total and 2 priority attention.
- [ ] 4.12 Calibrate to GREEN using only generic thresholds: if healthy silence fails, raise `RatioStandaloneMinHours` (per design open question), never touch per-meter behavior and never add meter ids or dates to engine code. Record final values in `DefaultConfig()`.
- [ ] 4.13 RED then GREEN: `backend/internal/engine/guard_test.go` `TestEngineHasNoMeterLiterals` scans non-test `backend/internal/engine/*.go` (comments included) for regex `M-\d+` and fails on a match; also asserts the package source contains no dataset dates (`2026-09-`).
- [ ] 4.14 Create `docs/thresholds.md` draft: one row per `Config` field with default, healthy-envelope evidence, and per-meter justification (this is where meter ids are allowed). Finalized into README in slice 9.
- [ ] 4.15 REFACTOR + verify: `go vet ./...`, `go test ./... -count=1`, `rg -n "expected[_]results" backend` returns nothing.

---

## Slice 5: Analysis jobs, anomalies API, status derivation

Branch: `feat/05-analysis-jobs-api` (base slice 4b branch). Forecast: ~520 lines (`size:exception` accepted). Task count: 14.
Spec coverage: ai-analysis (all except Template explainer body), meter-catalog (Derived meter status), dashboard-summary (Aggregate confidence, Latest analysis wins, After the demo analysis), platform-delivery (Configurable stage delay, single-flight and job-state tests, endpoint coverage).

- [ ] 5.1 RED: `backend/internal/platform/uuid_test.go`: `NewID` returns RFC 4122 v4 format, unique across 1,000 calls.
- [ ] 5.2 GREEN: `backend/internal/platform/uuid.go` (`crypto/rand`); add `AnalysisRepository`, `AnomalyRepository`, `AnomalyFilter`, `IDGenerator` to `backend/internal/app/ports.go`.
- [ ] 5.3 RED: `backend/internal/adapters/sqlite/analyses_test.go`: `Create`; `UpdateProgress`; `Complete` inserts anomalies and sets `done`/progress 100 in one tx (failure inside leaves no anomalies); `Fail`; `Get` returns `ErrNotFound`; `Latest`; `LatestDone` ignores failed and running; `FailInterrupted` marks pending/running as failed with `interrupted by restart`; anomaly `List` ordered by rank with `type`/`severity`/`meter_id` filters; `Get` restores evidence and related event.
- [ ] 5.4 GREEN: `backend/internal/adapters/sqlite/{analyses.go,anomalies.go}`.
- [ ] 5.5 RED: `backend/internal/app/summary_test.go`: 4 anomalies with 2 HIGH REAL/DQ gives `4 anomalies detected · 2 require priority attention`; zero gives `0 anomalies detected · 0 require priority attention`; singular forms for 1; FALSE_POSITIVE counted in total but not in priority.
- [ ] 5.6 GREEN: `backend/internal/app/summary.go` (uses U+00B7).
- [ ] 5.7 RED: `backend/internal/app/analysis_service_test.go` (fake repos, delay 0): two concurrent `Start` calls return the same id and one row; new id after done; new run after failed; stage order is the 7 stages with progress 0,14,28,42,57,71,85 then 100 (non-decreasing); `done` only at 100; engine error gives `failed` with non-empty error and zero anomalies visible; panic recovered into `failed`; anomalies become visible only at `done`; with delay > 0 at least two distinct stages observed while polling.
- [ ] 5.8 GREEN: `backend/internal/app/analysis_service.go` (single-flight by mutex + `activeID`, `context.WithoutCancel`, `StageDelay` option, explainer called for reason/action, summary persisted).
- [ ] 5.9 RED: extend `meter_service_test.go` and `dashboard_service_test.go`: status Critical for REAL HIGH, Alert for DATA_QUALITY/EXPLAINABLE/non-HIGH REAL, OK for FALSE_POSITIVE only or none; `aggregate_confidence` of 0.9, 0.8, 0.7, 0.6 is 0.75; null with no anomalies; failed newer analysis does not replace older done figures; `last_analysis` present after done.
- [ ] 5.10 GREEN: replace `NoAnalysis` with a repo-backed `MeterStatusSource` in `backend/internal/app/meter_service.go`; latest-done logic in `backend/internal/app/dashboard_service.go`.
- [ ] 5.11 RED: `backend/internal/adapters/http/analysis_http_test.go`: `POST /api/ai/analyze` 202 `{id}` and repeat returns same id while active; `GET /api/ai/analysis/{id}` done payload (status, stage, progress, started_at, finished_at, error, summary) and 404; `GET /api/ai/analysis/latest` 204 when none; `GET /api/anomalies` empty 200 before analysis, 4 rows ordered M-109, M-112, M-104, M-106 after a run on the demo dataset seeded from `data/*.csv`, `?type=DATA_QUALITY` returns only M-112, unknown `analysis_id` 404, invalid filter 400; `GET /api/anomalies/{id}` includes evidence and UNKNOWN event as context, 404 for unknown; meters list after the run shows M-109 Critical, M-112 and M-104 Alert, others OK; dashboard after run shows 4 / 2 / summary text; all routes 401 without token.
- [ ] 5.12 GREEN: `backend/internal/adapters/http/{handlers_analysis.go,handlers_anomalies.go}` and DTO additions in `dto.go` (`priority_rank`, `confidence_label`, `related_event` nullable, `evidence[]`).
- [ ] 5.13 RED then GREEN: boot recovery test (seed an analysis `running`, start wiring, it becomes `failed`, a new analysis can start) and wire in `backend/cmd/api/main.go`: `FailInterrupted` before serving, build `AnalysisService` with `ANALYSIS_STAGE_DELAY_MS`, explainer adapter, uuid generator.
- [ ] 5.14 REFACTOR + verify: `go vet ./...`, `go test ./... -count=1`; run the harness (login, analyze, poll to done, list anomalies).

---

## Slice 6: Web shell, login, dashboard

Branch: `feat/06-web-shell-dashboard` (base slice 5 branch). Forecast: ~450 lines (edge). Task count: 13.
Spec coverage: web-console (Login and session guard, Dashboard, Loading/error/empty states). Style: SaaS energy-management console per design section 6.5; atomic design, container (`*Page`) / presentational (`*View`).

- [ ] 6.1 Scaffold `frontend/{package.json,vite.config.ts,tsconfig.json,index.html,tailwind.config.ts,postcss.config.js}` (Vite 5, React 18.3, TS strict, React Router 6, TanStack Query 5, Recharts 2, Tailwind 3.4, Vitest + Testing Library) and `frontend/src/main.tsx`; `server.proxy['/api'] -> http://localhost:8080`.
- [ ] 6.2 RED: `frontend/src/lib/format.test.ts` (kWh with thousands separator and 1 decimal, percent with sign, date formats, `tabular-nums` helper not needed) and `frontend/src/lib/status.test.ts` (severity/type/status to token name). Confirm vitest runs and fails.
- [ ] 6.3 GREEN: `frontend/src/lib/{format.ts,status.ts}`.
- [ ] 6.4 Add `frontend/src/styles/{tokens.css,index.css}` with all design section 6.5 tokens (navy sidebar, teal primary, amber accent, severity/type/status colors, Inter font, 12px radius) and map them in `tailwind.config.ts`.
- [ ] 6.5 Atoms in `frontend/src/components/atoms/{Button,Card,Badge,Input,Select,Spinner,ProgressBar,Skeleton}.tsx` (props only, no hooks).
- [ ] 6.6 Molecules in `frontend/src/components/molecules/{KpiCard,SeverityBadge,TypeBadge,StatusPill,ConfidenceMeter}.tsx`.
- [ ] 6.7 RED then GREEN: `frontend/src/api/client.test.ts` and `frontend/src/api/client.ts`: sends `Authorization: Bearer`, maps error bodies to `ApiError`, on 401 clears the token and calls the unauthorized handler, network failure maps to `ApiError` with code `network`.
- [ ] 6.8 Add `frontend/src/api/{types.ts,auth.ts,dashboard.ts,analysis.ts}` (DTO types use the spec field names; `analysis.ts` exposes `startAnalysis` only in this slice).
- [ ] 6.9 Add `frontend/src/hooks/{useAuth.ts,useDashboard.ts}`, `frontend/src/app/{providers.tsx,AuthGuard.tsx,router.tsx}` (guard redirects to `/login`; token in localStorage; logout).
- [ ] 6.10 Organisms `frontend/src/components/organisms/AppShell.tsx` (Sidebar, Topbar with logout).
- [ ] 6.11 `frontend/src/pages/Login/{LoginPage.tsx,LoginView.tsx}` (demo credentials hint, error message on wrong credentials, stays on login).
- [ ] 6.12 `frontend/src/pages/Dashboard/{DashboardPage.tsx,DashboardView.tsx}`: KPIs (meters, total kWh, anomalies, high priority, aggregate confidence, last analysis), summary text, empty state inviting analysis when none completed, loading skeleton, error with Retry, **Run AI Analysis** trigger (container passes `onRun` calling `startAnalysis`; progress UI arrives in slice 8).
- [ ] 6.13 Verify: `cd frontend && npx vitest run --reporter=dot && npm run build`; harness: API + `npm run dev`, login with `admin/admin123`, dashboard shows KPIs.

---

## Slice 7: Meters list and meter detail

Branch: `feat/07-meters-detail` (base slice 6 branch). Forecast: ~430 lines (edge). Task count: 10.
Spec coverage: web-console (Meters list, Meter detail charts, Loading/error/empty states).

- [ ] 7.1 Add `frontend/src/api/meters.ts` (list with `q`, `status`, `sort`, `order`; detail; readings; events) and extend `frontend/src/api/types.ts`.
- [ ] 7.2 Add `frontend/src/hooks/{useMeters.ts,useMeterReadings.ts}` (query keys `['meters', params]`, `['meter', id]`, `['readings', id]`, `['events', id]`; parallel fetch).
- [ ] 7.3 Molecule `frontend/src/components/molecules/SearchFilterBar.tsx` (search input, status select, sort select; props only).
- [ ] 7.4 Organism `frontend/src/components/organisms/MetersTable.tsx` (id, total kWh, daily variation, status pill, row link to `/meters/:meterId`).
- [ ] 7.5 `frontend/src/pages/Meters/{MetersPage.tsx,MetersView.tsx}`: filter state in the URL search string, loading/error/empty states; typing `109` leaves only M-109; filter `Critical` and sort by variation supported.
- [ ] 7.6 RED: `frontend/src/lib/chartData.test.ts`: merges readings with baseline per metric, maps anomaly windows to `ReferenceArea` ranges, maps events to `ReferenceLine` markers, `window +- 24h` zoom helper.
- [ ] 7.7 GREEN: `frontend/src/lib/chartData.ts` and organism `frontend/src/components/organisms/MetricChart.tsx` (Recharts `ComposedChart`, observed solid line, baseline dashed line, severity-tinted `ReferenceArea`, event `ReferenceLine`).
- [ ] 7.8 Organisms `frontend/src/components/organisms/{ChartGrid.tsx,EventTimeline.tsx}` (four charts: kWh with baseline overlay, voltage, current, power factor).
- [ ] 7.9 `frontend/src/pages/MeterDetail/{MeterDetailPage.tsx,MeterDetailView.tsx}`: header with status, 4 charts, events, not-found state for unknown meter, error with Retry, **Run AI Analysis** trigger reusing the slice 6 mutation.
- [ ] 7.10 Verify: `npx vitest run --reporter=dot && npm run build`; harness: open M-109 detail, kWh rise visible from 2026-09-12 14:00 against baseline; unknown meter shows not-found.

---

## Slice 8: Run analysis, anomalies, investigation

Branch: `feat/08-analysis-anomalies-investigation` (base slice 7 branch). Forecast: ~480 lines (edge). Task count: 10.
Spec coverage: web-console (Run AI Analysis with live progress, Anomalies table, Investigation and action, Loading/error/empty states, Dashboard after analysis).

- [ ] 8.1 RED: `frontend/src/hooks/useAnalysisJob.test.ts`: `refetchInterval` predicate returns 500 for `pending`/`running` and `false` for `done`/`failed`/undefined; on `done` the hook invalidates `dashboard`, `meters`, `anomalies`, `meter`; trigger is disabled while active.
- [ ] 8.2 GREEN: `frontend/src/hooks/useAnalysisJob.ts` and `getAnalysis` in `frontend/src/api/analysis.ts` (mutation stores id, polls `['analysis', id]` every 500 ms).
- [ ] 8.3 Molecule `frontend/src/components/molecules/StageStepper.tsx` (7 named stages) and organism `frontend/src/components/organisms/AnalysisProgressPanel.tsx` (progress bar, current stage, link to Anomalies on `done`, error plus Retry on `failed`).
- [ ] 8.4 Wire `useAnalysisJob` and the panel into `frontend/src/pages/Dashboard/DashboardPage.tsx` and `frontend/src/pages/MeterDetail/MeterDetailPage.tsx`, replacing the slice 6 placeholder handler; trigger disabled with visible progress while a run is active.
- [ ] 8.5 Add `frontend/src/api/anomalies.ts` and `frontend/src/hooks/useAnomalies.ts` (filters `type`, `severity`, `meter_id`; detail by id).
- [ ] 8.6 Organism `frontend/src/components/organisms/AnomaliesTable.tsx` and `frontend/src/pages/Anomalies/{AnomaliesPage.tsx,AnomaliesView.tsx}`: rows ordered by priority (meter id, type badge, severity badge, confidence, investigate link), type and severity filters, summary banner, empty state offering to run analysis.
- [ ] 8.7 Molecule `frontend/src/components/molecules/EvidenceRow.tsx`, organisms `frontend/src/components/organisms/{EvidenceTable.tsx,ActionCard.tsx}` (metric, baseline, observed, delta, unit; recommended action card).
- [ ] 8.8 `frontend/src/pages/Investigation/{InvestigationPage.tsx,InvestigationView.tsx}`: type, severity, confidence meter, reason, evidence table, affected window, window chart zoomed to `window +- 24h`, related event (with "no explanation" note for UNKNOWN), **Recommended action** card.
- [ ] 8.9 Add a meter anomalies section to `frontend/src/pages/MeterDetail/MeterDetailView.tsx` (uses `useAnomalies` with `meter_id`).
- [ ] 8.10 Verify: `npx vitest run --reporter=dot && npm run build`; harness: API + Vite dev, run analysis through visible stages to done, Anomalies shows 4 rows in order M-109, M-112, M-104, M-106, M-109 investigation shows REAL_ANOMALY, HIGH, evidence, action; M-106 shows FALSE_POSITIVE, LOW and the outage.

---

## Slice 9: Delivery (Docker, nginx, compose, README)

Branch: `feat/09-delivery` (base slice 8 branch). Forecast: ~380 lines (edge). Task count: 11.
Spec coverage: platform-delivery (Zero-config compose, Data and persistence, Documented configuration, README, Automated verification, Source-data boundaries).

- [ ] 9.1 `backend/Dockerfile`: `golang:1.23-alpine` build (`CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api`) then `alpine:3.20` with `wget`, non-root user, `EXPOSE 8080`.
- [ ] 9.2 `frontend/Dockerfile`: `node:20-alpine` (`npm ci && npm run build`) then `nginx:alpine` with `deploy/nginx.conf`.
- [ ] 9.3 `deploy/nginx.conf`: serve SPA with `try_files $uri /index.html`, proxy `/api/` to `http://api:8080`.
- [ ] 9.4 `docker-compose.yml`: `api` (env defaults from design section 7 with `ANALYSIS_STAGE_DELAY_MS=300`, healthcheck `wget -qO- localhost:8080/api/health`, `./data:/data:ro`, named volume `energy-db` at `/var/lib/energy`), `web` (`depends_on: api: condition: service_healthy`, ports `8080:80`); no `.env` needed.
- [ ] 9.5 `README.md` part 1: how to run (`docker-compose up`, local dev with `go run ./cmd/api` and Vite), demo credentials and the 5-10 minute demo flow, architecture (layers, dependency direction, `Explainer` port, incident model, async job stages), key decisions (SQLite, no LLM, stdlib router, stateless token, atomic design).
- [ ] 9.6 `README.md` part 2: threshold derivation table listing every `Config` field with default, data-derived rationale and the per-meter justifications (healthy envelope vs targets), sourced from `docs/thresholds.md`.
- [ ] 9.7 Remove `docs/thresholds.md` after its content is merged into `README.md` (single source of truth); update any link.
- [ ] 9.8 `README.md` part 3: testing section with `go test ./... -count=1` (short summary output) and frontend test command, plus the environment variable table with defaults (stage delay 300 in compose, 0 in tests).
- [ ] 9.9 Source-data boundary check: `rg -n "expected[_]results" . --glob '!openspec/**'` returns nothing; confirm build, tests, and runtime read only `data/readings.csv` and `data/events.csv`.
- [ ] 9.10 Compose smoke (requires Docker): `docker compose config -q`; `docker compose up --build -d`; login on `http://localhost:8080`, 12 meters listed, run analysis to done with 4 anomalies; `docker compose down` then `up` keeps counts; `docker compose down -v` then `up` reseeds.
- [ ] 9.11 Final verification: `cd backend && go test ./... -count=1`, `cd frontend && npx vitest run --reporter=dot && npm run build`; tick the change success criteria checklist in `proposal.md` evidence notes if maintainers want it.

---

## Task Totals

| Slice | Branch | Tasks | Forecast lines |
|-------|--------|-------|----------------|
| 0 | `main` | 2 | 0 |
| 1 | `feat/01-backend-foundation` | 16 | ~420 |
| 2 | `feat/02-read-api-auth` | 17 | ~560 |
| 3 | `feat/03-engine-detectors` | 17 | ~520 |
| 4a | `feat/04a-engine-correlate-classify` | 4 | ~330 |
| 4b | `feat/04b-engine-score-dataset` | 11 | ~390 |
| 5 | `feat/05-analysis-jobs-api` | 14 | ~520 |
| 6 | `feat/06-web-shell-dashboard` | 13 | ~450 |
| 7 | `feat/07-meters-detail` | 10 | ~430 |
| 8 | `feat/08-analysis-anomalies-investigation` | 10 | ~480 |
| 9 | `feat/09-delivery` | 11 | ~380 |
| Total | | 125 | ~4,480 |

Sequential by design: slices are a stacked chain (each depends on the previous). Parallelizable inside a slice: RED tests for independent detectors (3.4/3.6/3.8/3.10/3.12/3.14), atoms vs molecules (6.5/6.6), Dockerfiles (9.1/9.2/9.3). Frontend slices 6-8 could start after slice 2 API contracts are stable but stay sequential in the stack.
