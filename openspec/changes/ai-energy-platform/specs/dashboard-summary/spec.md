# Dashboard Summary Specification

## Purpose

Provide the KPIs shown on the dashboard from a single authenticated endpoint.

## Requirements

### Requirement: KPI payload

`GET /api/dashboard/summary` MUST return: `meters_count`, `total_kwh`, `anomalies_count`, `high_priority_count`, `aggregate_confidence`, `last_analysis` (id, status, finished_at; nullable) and `summary_text` (nullable). Anomaly figures MUST come from the latest `done` analysis.

#### Scenario: Before any analysis

- GIVEN the seeded database and no completed analysis
- WHEN the client calls the endpoint
- THEN `meters_count` is 12 and `total_kwh` equals the sum of all readings
- AND `anomalies_count` and `high_priority_count` are 0
- AND `aggregate_confidence`, `last_analysis` and `summary_text` are null

#### Scenario: After the demo analysis

- GIVEN the demo analysis completed
- WHEN the client calls the endpoint
- THEN `anomalies_count` is 4 and `high_priority_count` is 2
- AND `summary_text` is `4 anomalies detected · 2 require priority attention`
- AND `last_analysis` references that analysis

### Requirement: Aggregate confidence

`aggregate_confidence` MUST be the mean confidence of the latest analysis's anomalies, and null when there are none.

#### Scenario: Mean confidence

- GIVEN anomalies with confidence 0.9, 0.8, 0.7 and 0.6
- WHEN the summary is computed
- THEN `aggregate_confidence` is 0.75

### Requirement: Latest analysis wins

A newer completed analysis MUST replace the previous one as the source of anomaly figures. A newer failed analysis MUST NOT.

#### Scenario: Failed rerun

- GIVEN a completed analysis A and a later failed analysis B
- WHEN the client calls the endpoint
- THEN the anomaly figures come from A
