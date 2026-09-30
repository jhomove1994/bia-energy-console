# Proposal: AI Energy Management Platform (MVP)

## Intent

Build an end-to-end MVP that turns 14 days of hourly readings from 12 electrical meters into operational decisions. The system detects anomalies, explains them with numeric evidence, prioritizes them, and recommends an action. A deterministic, traceable engine in Go decides, using rules plus robust statistics (median/MAD, robust z-score per hour of day). The brief explicitly allows rules, statistics, Z-score, Isolation Forest, time series, ML, LLM, or a combination. Explanations are rendered from the evidence by a template explainer, so there is no external AI service and no API key. The demo must run through `docker-compose` with zero configuration.

Success means an evaluator can follow Login -> Dashboard -> Meters -> M-109 -> Run AI Analysis -> Anomalies -> Investigation -> Action in 5-10 minutes. The four validation cases must come out of generic rules, not hardcoded meter IDs.

## Data Findings (from exploration, verified)

| Meter | Evidence | Expected -> Supported |
|---|---|---|
| Healthy (8 meters) | Hourly dev vs hour-of-day median at most 17%; daily kWh about ±2%; no V jump >8 V; no current jump >50%; PF 0.87-0.99; ratio `kWh/(V*A*PF/1000)` median 1.02-1.12, MAD 0.03-0.05 | Must stay silent |
| M-109 | From 09-12 14:00: kWh +118-130%/h, daily +109%, current 195->410 A, PF 0.94->0.74, V -2-3 V, ratio 1.07->1.32-1.43, 58 h >=40%. Event `UNKNOWN` | REAL_ANOMALY / HIGH / top priority -> Yes |
| M-106 | 09-08 00:00-11:00: kWh -80%, current 39-61 A, V/PF normal, recovers at 12:00. `SCHEDULED_OUTAGE` (12 h) at 09-08 00:00 | FALSE_POSITIVE / LOW -> Yes |
| M-104 | From 09-11 00:00: daily +47%, current 213->316 A, PF 0.91->0.89, V and ratio stable. `OPERATIONAL_CHANGE` at the step | EXPLAINABLE_ANOMALY / MEDIUM -> Yes |
| M-112 | Daily kWh stable (±1%). On 09-13/14 every 3 h: V toggles about 202/240 V, PF quantized 0.58/0.72/0.98, current 46-218 A, ratio 0.58-4.22. 32 V jumps >8 V, 13 current jumps >50%. `DATA_QUALITY` event | DATA_QUALITY / HIGH -> Yes |

**Nuances to respect:**
- M-109 also has one current jump (+110%), one PF drop, and a ratio shift. DATA_QUALITY rules must require repeated, oscillating jumps (reversals), not a single step. Otherwise M-109 would be misclassified.
- M-112 must reach DATA_QUALITY/HIGH from the reading rules alone. The `DATA_QUALITY` event may only raise confidence.
- The M-109 event record exists but its type is `UNKNOWN`, so it has zero explanatory power.
- The dataset has no gaps and `status=OK` on every row. The gap and status detectors must exist, but they produce nothing here.
- The brief's example figures (M-109 +103.7%, baseline 1,070) are illustrative. Our numbers come from the data (about +109%).
- "4 anomalías detectadas · 2 requieren atención prioritaria" counts all four findings, including the FALSE_POSITIVE. Priority attention = HIGH severity AND type in {REAL, DATA_QUALITY}.

## Scope

### In Scope
- Go backend (hexagonal): CSV seed at startup, SQLite persistence, REST API (minimum set + login), async analysis jobs.
- A deterministic anomaly engine (Readings -> Baseline -> Detection -> Correlation -> Events -> Explanation -> Recommendation) with scores and evidence.
- An `Explainer` port with a template adapter only. It builds `reason` and `recommended_action` from the evidence.
- React + TypeScript SaaS console: Login, Dashboard, Meters (filter/search/sort), Meter Detail (kWh/V/A/PF charts with baseline), Run AI Analysis with live progress, Anomalies, Investigation.
- `docker-compose` (api + web), a README with architecture, decisions and the derivation of each threshold.
- Engine unit tests (per-rule synthetic tests plus a dataset test for the 4 cases), basic API tests, short `go test` output.

### Out of Scope
- Real identity management (multi-user, roles, password reset, OAuth).
- Streaming or real-time ingestion, CSV upload UI, multi-tenant.
- ML model training, including Isolation Forest and forecasting.
- Anomaly workflow beyond read-only status (no assignment or ticketing).
- Any use of `expected_results.csv`.
- LLM integration. It would force evaluators to supply an API key, and the brief accepts rules and statistics. The `Explainer` port keeps it pluggable later.

## Capabilities

### New Capabilities
- `data-ingestion`: CSV loading at startup into SQLite (idempotent), and validation of the reading cadence and status.
- `meter-catalog`: the meters list and detail, readings series, derived variation and meter status (OK/Alert/Critical).
- `anomaly-engine`: baseline, detectors, event correlation, classification, severity/confidence/priority scoring, recommendation, evidence.
- `ai-analysis`: async analysis jobs (pending/running/done/failed with stage progress), the anomalies API, the template explainer behind the `Explainer` port.
- `dashboard-summary`: KPIs (meters, total kWh, anomalies, high priority, aggregate confidence, last analysis).
- `auth-session`: a single demo login that issues a token, and middleware protecting `/api/*`.
- `web-console`: the React SaaS UI covering the full demo flow.
- `platform-delivery`: docker-compose, Dockerfiles, the README with architecture, decisions and threshold rationale.

### Modified Capabilities
- None (greenfield).

## Approach

### Decisions

| Area | Decision | Why |
|---|---|---|
| Persistence | **SQLite** via `modernc.org/sqlite` (pure Go, no CGO) | 4,032 rows, a single process, zero ops, a one-file volume, a trivial Docker build. PostgreSQL adds a service and a readiness race for no benefit at this size. Repositories sit behind ports, so a swap stays cheap. |
| Go toolchain | `go 1.23` in `go.mod` (plus the `toolchain` directive), Docker `golang:1.23-alpine` | Gives the stdlib `http.ServeMux` method/path patterns (Go 1.22+) with no router dependency, and current driver versions. Local 1.21.5 auto-downloads the toolchain (`GOTOOLCHAIN=auto`). |
| Architecture | Hexagonal: `domain` <- `engine` (pure) <- `app` (use cases + ports) <- `adapters` (http, sqlite, csv, explain) | The engine is pure functions over slices. It is testable without IO, and the rules are traceable. |
| Frontend | Vite + React + TS, React Router, TanStack Query (polling), Recharts, Tailwind | A fast SaaS look and native chart and polling support. |
| Serving | nginx serves the SPA and proxies `/api` to the Go service | One origin and no CORS in the demo. |
| AI layer | Rules plus robust statistics decide everything. The `Explainer` port renders `reason` and the action from evidence via templates. No LLM adapter. | Zero config for evaluators (no API key), fully deterministic and testable. The brief allows a rules/statistics approach. |

### Backend layout

```
backend/
  cmd/api/main.go                 # wiring: config, sqlite, seed, http
  internal/domain/                # Meter, Reading, Event, Anomaly, Analysis, Evidence, enums
  internal/engine/                # baseline.go detect.go quality.go correlate.go classify.go score.go recommend.go config.go
  internal/app/                   # MeterService, AnalysisService (job runner), DashboardService, ports.go
  internal/adapters/sqlite/       # repos + migrations
  internal/adapters/csvload/      # readings/events seed
  internal/adapters/http/         # router, handlers, DTOs, auth middleware
  internal/adapters/explain/      # template.go, llm.go
frontend/                         # Vite app (pages, components, api client)
data/                             # readings.csv, events.csv (mounted read-only)
docker-compose.yml, README.md
```

### Data model (SQLite)

| Table | Key columns |
|---|---|
| `meters` | id, meter_id UNIQUE, created_at (no name or location: the CSV lacks them and we do not invent data) |
| `readings` | id, meter_id, ts, consumption_kwh, voltage_v, current_a, power_factor, status; UNIQUE(meter_id, ts) |
| `events` | id, meter_id, ts, type, description |
| `analyses` | id (uuid), status, stage, progress 0-100, started_at, finished_at, error, summary_json |
| `anomalies` | id, analysis_id, meter_id, detected_at, window_start, window_end, kind, type, severity, confidence, priority_score, reason, recommended_action, related_event_id, evidence_json, status |

Meter status is derived from the latest analysis: Critical = REAL HIGH; Alert = DATA_QUALITY or EXPLAINABLE; OK = none or FALSE_POSITIVE.

### Engine thresholds (configurable in `engine/config.go`, documented in README)

| Parameter | Candidate | Data evidence |
|---|---|---|
| Baseline | Per meter x hour-of-day **median + MAD** over the first 7 days of the series (the window is relative to data start, not a hardcoded date) | Day/night shape 23 vs 36 kWh. All incidents start on day 8 or later. The median resists contamination. |
| MAD floor | `max(MAD, 3% of median)` | Tiny MAD made robust-z >3.5 fire 6-33 times per healthy meter |
| Hourly effect gate | \|dev\| >= **30%** AND robust-z >= 3.5 | Healthy max 17% (a margin of about 1.8x). Targets are 47-130%. |
| Persistent shift | >= **6** consecutive flagged hours, same sign | Healthy streak 0. M-106 12 h, M-109 58 h, M-104 >= 4 days |
| Spike / outlier | 1-5 flagged hours, or one point >= 60% | Healthy none >= 40% |
| Daily variation | \|daily vs baseline daily\| >= **25%** | Healthy ±2% |
| Hourly pattern | Day profile correlation vs baseline profile < 0.8 (**to calibrate in design** on healthy meters) | Not profiled yet |
| Recovery | The last 12 h are back inside the 30% gate -> transient | M-106 recovers at 12:00, M-104 and M-109 do not |
| V jump (DQ) | \|ΔV\| hour-to-hour > **8 V** | Healthy none. M-112 about 38 V. M-109 2-3 V. |
| Current jump (DQ) | \|ΔA\|/A > **50%** | Healthy none. M-112 13. M-109 exactly 1 (a step). |
| PF jump (DQ) | \|ΔPF\| >= 0.15 | M-112 quantized 0.58/0.72/0.98. M-109 has one step. |
| DQ oscillation | >= **3** jumps with sign reversals within 24 h | Separates M-112 (oscillating) from M-109 (single step) |
| Ratio deviation | \|ratio - meter median\| / median >= **20%** | Healthy MAD about 4%. M-109 +23-34% and stable (electrical corroboration). M-112 0.58-4.22 and unstable (DQ). |
| Gaps / status | Missing hourly timestamp or status != OK -> DQ evidence | None in this dataset |
| Event match window | Event ts within **±2 h** of the anomaly onset | All three events match their onset exactly |

**Event type -> explanatory power** (direction-aware):

| Event type | Explains | Power |
|---|---|---|
| `SCHEDULED_OUTAGE` / maintenance | Decreases only | 1.0 |
| `OPERATIONAL_CHANGE` | Persistent shifts (either sign) | 0.9 |
| `DATA_QUALITY` | Nothing in consumption. Adds DQ confidence. | 0 (+0.1 DQ conf) |
| `UNKNOWN` / unrecognized | Nothing | 0 |

### Classification (ordered rules)

1. DQ oscillation OR ratio instability OR gaps/status -> **DATA_QUALITY**.
2. A consumption anomaly with a matching event whose power is >= 0.7 and whose direction is compatible -> recovered: **FALSE_POSITIVE**. Persistent: **EXPLAINABLE_ANOMALY**.
3. Otherwise -> **REAL_ANOMALY**.

### Scoring

| Score | Formula | Bands / check |
|---|---|---|
| Severity (consumption), 0-100 | 40·min(\|daily var\|/100%,1) + 20·min(hours/24,1) + corroboration (20 if >=2 of {ΔA>=30%, ΔPF>=0.1, ratio>=20%}, 10 if 1) + 20 if unexplained. EXPLAINABLE is capped at MEDIUM. FALSE_POSITIVE is forced to LOW. | HIGH >=70, MEDIUM 40-69, LOW <40. M-109 about 100 HIGH. M-104 about 49 MEDIUM. M-106 about 36 LOW. |
| Severity (DQ), 0-100 | 50·min(jumps/10,1) + 30·min(dq_hours/24,1) + 20·(signal kinds/3) | M-112 about 93 HIGH (no event needed) |
| Confidence, 0-1 | 0.5 + 0.2·min((\|dev\|-30%)/30%,1) + 0.15·min(h/24,1) + 0.1·corroboration/3 + 0.05·event clarity. DQ variant: 0.5 + 0.25·min(jumps/10,1) + 0.15·kinds/3 + 0.1 if a DQ event. Capped at 0.98. | Label High >=0.85, Medium 0.6-0.85, Low <0.6 |
| Priority | sevWeight(H3/M2/L1) x confidence x typeWeight(REAL 1.0, DQ 0.9, EXPL 0.6, FP 0.2) | Order: M-109 > M-112 > M-104 > M-106 |

Every anomaly returns `meter_id, anomaly, type, severity, confidence, reason, recommended_action, evidence[]`. Each evidence item has: metric, baseline, observed, delta %, unit, and the related event if any.

### Async analysis job

`POST /api/ai/analyze` inserts `analyses(pending)` and returns `202 {id}`. If a run is already active, it returns the same id (single-flight). A goroutine runs the 7 stages, updates `stage/progress`, and persists anomalies and the summary atomically at `done`. On error it sets `failed` with an error message. The UI polls `GET /api/ai/analysis/:id` every 500 ms. An optional per-stage delay (`ANALYSIS_STAGE_DELAY_MS`, default 300) keeps progress visible in the demo. Runs left in `running` at boot are marked `failed`.

## Delivery Plan (work-unit slices, 400-line budget)

| # | Slice (one PR) | Contents incl. tests/docs | Forecast lines |
|---|---|---|---|
| 1 | Backend foundation | go module, domain, sqlite adapter + migrations, CSV seed, startup wiring, tests | ~350 |
| 2 | Read API + auth | meters, readings, dashboard summary (pre-analysis), login + middleware, httptest suite | ~380 |
| 3 | Engine: baseline + detectors | baseline, consumption and DQ detectors, config, synthetic unit tests | ~400 |
| 4 | Engine: correlate + classify + score | correlation, classification, scoring, recommendations, template explainer, 4-case dataset test, threshold doc | ~400 |
| 5 | Analysis jobs + anomalies API | job runner, analyses/anomalies repos, 4 endpoints, tests | ~380 |
| 6 | Web shell + Dashboard | Vite/TS setup, layout and design tokens, login, API client, dashboard | ~400 |
| 7 | Meters + Detail | list with filter/search/sort, detail with charts vs baseline | ~400 |
| 8 | Analysis + Anomalies + Investigation | Run AI Analysis progress, anomalies table, investigation and action view | ~400 |
| 9 | Delivery | Dockerfiles, nginx, docker-compose, README (architecture, decisions, thresholds) | ~300 |

**Total about 3,400 authored lines (lockfiles and CSV data excluded).**
- 400-line budget risk: High.
- Chained PRs recommended: Yes.
- Decision needed before apply: Yes (chain strategy).

Slices 3, 4, 6, 7 and 8 sit at the budget edge. If one honestly exceeds it after a single slicing pass, report it with `size:exception` rather than shrinking the code.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/` | New | Go API, engine, adapters, tests |
| `frontend/` | New | React SaaS console |
| `data/` | New | Copies of `docs/readings.csv` and `docs/events.csv` mounted into the api |
| `docker-compose.yml`, `README.md` | New | Runtime and documentation |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Local Go 1.21.5 cannot build a `go 1.23` module offline (toolchain auto-download needs network) | Med | `GOTOOLCHAIN=auto` or `brew upgrade go`. Docker builds are unaffected. |
| The latest `modernc.org/sqlite` needs a newer Go than the pinned toolchain | Low | Pin a driver version compatible with go 1.23 in `go.mod` |
| M-109 misread as DATA_QUALITY (single current/PF/ratio step) | Med | Require oscillation (>=3 reversals in 24 h) for DQ, with a dedicated test |
| Overfitting thresholds to the 4 targets | Med | Every threshold is tied to the healthy envelope, and a test asserts the 8 healthy meters produce zero anomalies |
| Hourly-pattern detector is uncalibrated | Med | Calibrate on healthy meters in design. Keep it evidence-only if it is noisy. |
| Parsing event duration from free text ("for 12 hours") is fragile | Low | Use recovery detection instead of text parsing. The duration regex is optional evidence only. |
| Repo has no commits and the branch is `master` while main is `main` | Med | Create a baseline commit on `main` before the first slice |

## Rollback Plan

The change is greenfield and each slice is an independent PR. Revert the offending slice PR (or drop it from the chain) and the earlier slices remain functional. The SQLite file is disposable: delete the volume and the data reseeds from CSV at startup.

## Dependencies

- Go 1.23 toolchain, `modernc.org/sqlite`, `github.com/google/uuid` (or stdlib crypto/rand).
- Node 20+, Vite, React 18/19, React Router, TanStack Query, Recharts, Tailwind.
- Docker / docker-compose.

## Resolved Decisions (user review, 2026-09-30)

1. Go toolchain: `go 1.23` in `go.mod` using the stdlib router. Local builds use `brew upgrade go` or `GOTOOLCHAIN=auto`.
2. LLM: none. Template explainer only, so evaluators need no API key. The approach is rules plus robust statistics, which the brief explicitly allows.
3. Auth: a single demo user from env. Login issues a stateless HMAC token, and middleware protects `/api/*`.
4. Delivery: `ask-on-risk` with chain strategy `stacked-to-main`. The base branch is `main`. Create a baseline commit on `main` (`.gitignore`, `docs/`, `openspec/`) before slice 1.
5. Meter name/location: not stored or synthesized. The UI shows `meter_id`.
6. Per-stage delay: `ANALYSIS_STAGE_DELAY_MS`, default 300 for the demo and 0 in tests, so the pending/running/done progress stays visible.
7. User-facing copy in Spanish (brief examples are Spanish); code/API/docs in English.

## Success Criteria

- [ ] `docker-compose up` serves the UI and API with no API keys. The data is seeded from CSV at startup.
- [ ] Analysis yields exactly: M-109 REAL/HIGH (priority 1), M-112 DATA_QUALITY/HIGH, M-104 EXPLAINABLE/MEDIUM, M-106 FALSE_POSITIVE/LOW. The 8 healthy meters produce no findings.
- [ ] The summary reads "4 anomalías detectadas · 2 requieren atención prioritaria".
- [ ] No meter_id literal or expected result appears in engine code (checked by grep in review).
- [ ] Every anomaly exposes numeric evidence, a reason and a recommended action. The UI shows them in Investigation.
- [ ] The analysis progresses pending -> running -> done with visible stage progress.
- [ ] The engine tests (4 cases + healthy silence + per-rule) and the API tests pass with short `go test` output.
- [ ] The README documents the architecture, the decisions and the derivation of each threshold.
