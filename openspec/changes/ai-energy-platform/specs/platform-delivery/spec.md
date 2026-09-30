# Platform Delivery Specification

## Purpose

Make the platform runnable and reviewable with one command, no keys, and documented decisions.

## Requirements

### Requirement: Zero-config compose

`docker-compose up` MUST build and start an `api` service and a `web` service with no `.env` file, no API keys and no manual steps. The web service MUST serve the SPA and proxy `/api` to the API on one origin. The dataset MUST be seeded at startup.

#### Scenario: Cold start

- GIVEN a clean checkout with Docker installed and no environment files
- WHEN the user runs `docker-compose up`
- THEN the UI is reachable on the documented port
- AND login with the documented demo credentials works
- AND the meters list shows 12 meters

#### Scenario: Analysis in the stack

- GIVEN the stack is up
- WHEN the user runs the analysis from the UI
- THEN it completes `done` with 4 anomalies

### Requirement: Data and persistence

Source CSVs MUST be mounted read-only into the API. SQLite MUST persist on a named volume. Removing the volume and restarting MUST reseed from CSV.

#### Scenario: Restart keeps data

- GIVEN the stack ran and was stopped
- WHEN it is started again
- THEN row counts are unchanged and no duplicates exist

#### Scenario: Reset

- GIVEN the volume is removed
- WHEN the stack starts
- THEN the database is reseeded from the CSVs

### Requirement: Documented configuration

The API MUST read demo credentials, token secret and lifetime, data directory, database path and `ANALYSIS_STAGE_DELAY_MS` from environment with documented defaults (stage delay 300 ms in compose, 0 in tests).

#### Scenario: Stage delay override

- GIVEN `ANALYSIS_STAGE_DELAY_MS=0`
- WHEN an analysis runs
- THEN it completes without artificial delay

### Requirement: README

The README MUST cover: how to run, demo credentials and demo flow, architecture (layers and the `Explainer` port), key decisions (SQLite, no LLM, stdlib router, stateless token), and the derivation of each engine threshold from the healthy-meter envelope.

#### Scenario: Threshold documentation

- GIVEN the README
- WHEN a reviewer looks up any engine parameter
- THEN its default and data-based rationale are listed

### Requirement: Automated verification

`go test ./...` MUST pass with short output and include: per-rule synthetic engine tests, the four validation cases, healthy-meter silence, single-flight and job-state tests, auth protection tests, and an API test for each minimum endpoint (success, 404 and response shape).

#### Scenario: Test run

- GIVEN a checkout with Go 1.23
- WHEN `go test ./...` runs
- THEN all packages pass

#### Scenario: Endpoint coverage

- GIVEN the API test suite
- WHEN endpoints are enumerated
- THEN login, meters, meter detail, readings, dashboard summary, analyze, analysis status, anomalies and anomaly detail each have at least one test

### Requirement: Source-data boundaries

The build, tests and runtime MUST NOT read or depend on any expected-results file.

#### Scenario: No expected results

- GIVEN the repository
- WHEN searched for `expected_results`
- THEN no code or configuration references it
