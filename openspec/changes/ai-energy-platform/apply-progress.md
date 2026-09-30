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

## Remaining
Slices 3 - 9 (90 tasks) not started.
