# Web Console Specification

## Purpose

A React + TypeScript SaaS console that lets an evaluator complete Login -> Dashboard -> Meters -> M-109 -> Run AI Analysis -> Anomalies -> Investigation -> Action in 5-10 minutes. Meters are identified by `meter_id` only.

## Requirements

### Requirement: Login and session guard

The console MUST show a login page, store the token on success, redirect unauthenticated visits to it, and return to it on any `401`. It MUST offer logout.

#### Scenario: Successful login

- GIVEN valid demo credentials
- WHEN the user submits the form
- THEN the dashboard opens

#### Scenario: Wrong credentials

- GIVEN wrong credentials
- WHEN the user submits
- THEN an error message shows and the user stays on login

#### Scenario: Expired session

- GIVEN a logged-in user whose token expired
- WHEN any API call returns `401`
- THEN the user is redirected to login

### Requirement: Dashboard

The dashboard MUST show meters, total kWh, anomalies, high-priority count, aggregate confidence, last analysis and the summary text, with an entry point to run analysis.

#### Scenario: After analysis

- GIVEN the demo analysis completed
- WHEN the user opens the dashboard
- THEN it shows `4 anomalies detected · 2 require priority attention`

#### Scenario: Before analysis

- GIVEN no analysis has completed
- WHEN the user opens the dashboard
- THEN the anomaly KPIs show an empty state that invites running the analysis

### Requirement: Meters list

The console MUST list meters with id, total kWh, daily variation and status badge, and support search by id, filter by status and sorting.

#### Scenario: Search

- GIVEN 12 meters
- WHEN the user types `109`
- THEN only M-109 remains

#### Scenario: Filter and sort

- GIVEN a completed demo analysis
- WHEN the user filters by `Critical`
- THEN only M-109 is listed
- AND sorting by daily variation reorders the list

### Requirement: Meter detail charts

The meter detail MUST show separate charts for kWh, voltage, current and power factor over time. The kWh chart MUST overlay the hour-of-day baseline.

#### Scenario: M-109 detail

- GIVEN the user opens M-109
- WHEN the page loads
- THEN four charts render, and kWh shows observed against baseline with the rise visible from 2026-09-12 14:00

#### Scenario: Unknown meter

- GIVEN the URL names a nonexistent meter
- WHEN the page loads
- THEN a not-found state is shown

### Requirement: Run AI Analysis with live progress

From the dashboard and from meter detail the user MUST be able to start an analysis. The console MUST poll status about every 500 ms, show stage and progress, disable the trigger while a run is active, and offer a link to anomalies on `done`. On `failed` it MUST show the error and allow retry.

#### Scenario: Happy path

- GIVEN the user clicks Run AI Analysis
- WHEN the run proceeds
- THEN progress advances visibly through stages until `done`
- AND a link to Anomalies appears

#### Scenario: Already running

- GIVEN an analysis is active
- WHEN the user views the trigger
- THEN it is disabled and shows current progress

#### Scenario: Failure

- GIVEN the analysis ends `failed`
- WHEN the console polls
- THEN the error is displayed with a Retry action

### Requirement: Anomalies table

The console MUST list anomalies ordered by priority with meter id, type, severity, confidence, and a link to investigation. Type and severity MUST be filterable.

#### Scenario: Demo table

- GIVEN the demo analysis completed
- WHEN the user opens Anomalies
- THEN four rows appear in order M-109, M-112, M-104, M-106

#### Scenario: Empty

- GIVEN no completed analysis
- WHEN the user opens Anomalies
- THEN an empty state offers to run analysis

### Requirement: Investigation and action

The investigation view MUST show type, severity, confidence, the reason, the numeric evidence list (metric, baseline, observed, delta, unit), the related event if any, the affected window and the recommended action.

#### Scenario: M-109 investigation

- GIVEN the user opens the M-109 anomaly
- WHEN the page loads
- THEN it shows REAL_ANOMALY, HIGH, the evidence list, the recommended action, and no explanatory event

#### Scenario: M-106 investigation

- GIVEN the user opens the M-106 anomaly
- WHEN the page loads
- THEN it shows FALSE_POSITIVE, LOW and the scheduled outage as related event

### Requirement: Loading, error and empty states

Every data view MUST show loading, error (with retry) and empty states rather than blank screens.

#### Scenario: API unreachable

- GIVEN the API returns a network error
- WHEN a view loads
- THEN an error message with Retry is shown
