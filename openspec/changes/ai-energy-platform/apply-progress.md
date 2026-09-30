# Apply Progress: ai-energy-platform

Mode: Strict TDD (runner `go test ./... -count=1`, cwd `backend`). Store: hybrid. Delivery: stacked-to-main, `size:exception` accepted for PRs 2, 3, 5.

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

## Remaining
Slices 2 - 9 (107 tasks) not started.
