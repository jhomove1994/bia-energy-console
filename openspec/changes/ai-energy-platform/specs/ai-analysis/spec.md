# AI Analysis Specification

## Purpose

Run the engine as an asynchronous, observable job and expose its anomalies, with explanations rendered from evidence by a template explainer (no external AI service, no API key). All endpoints require authentication.

## Requirements

### Requirement: Start analysis

`POST /api/ai/analyze` MUST create an analysis in `pending` state and return `202` with `{ "id": "<uuid>" }` without waiting for completion.

#### Scenario: Start

- GIVEN no analysis is active
- WHEN an authenticated client posts to `/api/ai/analyze`
- THEN the response is `202` with a new id
- AND the analysis is retrievable by that id

### Requirement: Single-flight

While an analysis is `pending` or `running`, further `POST /api/ai/analyze` calls MUST return `202` with the same id and MUST NOT start another run.

#### Scenario: Concurrent requests

- GIVEN an analysis is running with id A
- WHEN two more clients post to `/api/ai/analyze`
- THEN both receive `202` with id A
- AND only one analysis row exists

#### Scenario: New run after completion

- GIVEN analysis A is `done`
- WHEN a client posts to `/api/ai/analyze`
- THEN a new id B is returned

### Requirement: Job states and progress

An analysis MUST move `pending` -> `running` -> `done` or `failed`. It MUST expose `status`, the current `stage`, and `progress` (0-100, non-decreasing). Stages MUST cover the pipeline (readings, baseline, detection, correlation, events, explanation, recommendation). `done` MUST have progress 100.

#### Scenario: Observable progress

- GIVEN a stage delay of 300 ms
- WHEN a client polls `GET /api/ai/analysis/{id}` until finished
- THEN it observes `pending` or `running` with at least two distinct stage values
- AND progress never decreases
- AND the final status is `done` with progress 100

#### Scenario: Zero delay

- GIVEN a stage delay of 0
- WHEN an analysis runs
- THEN it reaches `done` with no artificial wait

### Requirement: Analysis status endpoint

`GET /api/ai/analysis/{id}` MUST return `id`, `status`, `stage`, `progress`, `started_at`, `finished_at`, `error` and, when `done`, a `summary` containing the anomaly count, priority-attention count and summary text. An unknown id MUST return `404`.

#### Scenario: Done payload

- GIVEN a completed demo analysis
- WHEN the client fetches it
- THEN status is `done`, error is empty and the summary is present

#### Scenario: Unknown id

- GIVEN no analysis with that id
- WHEN the client fetches it
- THEN the response is `404`

### Requirement: Failure handling and atomic results

If the engine or persistence errors, the analysis MUST end `failed` with a non-empty `error`, and no anomalies from that run MUST be visible. Anomalies and summary MUST become visible atomically at `done`. A failed run MUST NOT block a later run.

#### Scenario: Engine error

- GIVEN the engine fails mid-run
- WHEN the client fetches the analysis
- THEN status is `failed` with an error message
- AND `GET /api/anomalies` does not include partial results from that run

#### Scenario: Retry after failure

- GIVEN a failed analysis
- WHEN a client posts to `/api/ai/analyze`
- THEN a new analysis starts

### Requirement: Interrupted run recovery

At startup, any analysis left `pending` or `running` MUST be marked `failed` with an explanatory error.

#### Scenario: Restart during a run

- GIVEN the process stopped while analysis A was `running`
- WHEN the API restarts
- THEN A is `failed`
- AND a new analysis can be started

### Requirement: Summary text and priority attention

The summary text MUST read `"{N} anomalies detected · {P} require priority attention"`, where N counts all anomalies including FALSE_POSITIVE and P counts anomalies with HIGH severity and type REAL_ANOMALY or DATA_QUALITY.

#### Scenario: Demo summary

- GIVEN the demo analysis completed
- WHEN the summary is read
- THEN the text is exactly `4 anomalies detected · 2 require priority attention`

#### Scenario: No anomalies

- GIVEN an analysis with zero anomalies
- WHEN the summary is read
- THEN the text is `0 anomalies detected · 0 require priority attention`

### Requirement: Anomaly list

`GET /api/anomalies` MUST return the anomalies of the latest `done` analysis ordered by priority rank, each with: `id`, `analysis_id`, `meter_id`, `anomaly` (short title), `type`, `severity`, `confidence`, `confidence_label`, `priority_score`, `priority_rank`, `reason`, `recommended_action`, `window_start`, `window_end`, `related_event` (nullable) and `evidence[]`. It SHOULD accept `type`, `severity` and `meter_id` filters. With no completed analysis it MUST return an empty list.

#### Scenario: Demo list

- GIVEN the demo analysis completed
- WHEN the client calls `GET /api/anomalies`
- THEN 4 anomalies are returned in order M-109, M-112, M-104, M-106
- AND each has a non-empty `evidence` list, `reason` and `recommended_action`

#### Scenario: Before any analysis

- GIVEN no analysis has completed
- WHEN the client calls `GET /api/anomalies`
- THEN the response is `200` with an empty list

#### Scenario: Filter by type

- GIVEN the demo analysis completed
- WHEN the client calls `GET /api/anomalies?type=DATA_QUALITY`
- THEN only the M-112 anomaly is returned

### Requirement: Anomaly detail

`GET /api/anomalies/{id}` MUST return the full anomaly, including the evidence list. An unknown id MUST return `404`.

#### Scenario: Detail

- GIVEN the M-109 anomaly id
- WHEN the client fetches it
- THEN the response includes consumption and current evidence items and the unrelated `UNKNOWN` event only as context

#### Scenario: Unknown id

- GIVEN no anomaly with that id
- WHEN the client fetches it
- THEN the response is `404`

### Requirement: Template explainer

`reason` and `recommended_action` MUST be rendered by an `Explainer` port whose only shipped adapter is deterministic and template-based, building text from the anomaly's evidence. The system MUST NOT call any external AI service or require an API key.

#### Scenario: Reason cites evidence

- GIVEN the M-109 anomaly
- WHEN its reason is read
- THEN it cites the observed and baseline values and percentage change from the evidence
- AND it states that no event explains the change

#### Scenario: Deterministic and offline

- GIVEN no network access and no API key configured
- WHEN an analysis runs twice on the same data
- THEN both runs succeed with identical text
