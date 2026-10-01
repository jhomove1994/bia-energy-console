# Meter Catalog Specification

## Purpose

Expose meters, their readings and derived status through the read API. All endpoints require authentication (see `auth-session`).

## Requirements

### Requirement: List meters

`GET /api/meters` MUST return every meter with: `meter_id`, latest reading timestamp, total consumption (kWh), daily variation (%), and `status` (`OK`, `Alert` or `Critical`).

#### Scenario: List after seed

- GIVEN the seeded database
- WHEN an authenticated client calls `GET /api/meters`
- THEN the response contains 12 meters (M-101 to M-112) with the fields above
- AND no meter has a name or location field

### Requirement: Optional list filtering and sorting

The list endpoint SHOULD accept `q` (case-insensitive `meter_id` substring), `status` and `sort` parameters. Unknown `status` or `sort` values MUST yield `400`.

#### Scenario: Search by id

- GIVEN 12 seeded meters
- WHEN the client calls `GET /api/meters?q=m-10`
- THEN only meters whose id contains `M-10` are returned

#### Scenario: Invalid status filter

- GIVEN any state
- WHEN the client calls `GET /api/meters?status=BROKEN`
- THEN the response is `400`

### Requirement: Derived meter status

Meter status MUST derive from the latest completed analysis: `Critical` if it has a REAL_ANOMALY of HIGH severity; else `Alert` if it has a DATA_QUALITY or EXPLAINABLE_ANOMALY; else `OK` (including FALSE_POSITIVE only, or no findings). With no completed analysis, every meter MUST be `OK`.

#### Scenario: No analysis yet

- GIVEN no analysis has completed
- WHEN the client lists meters
- THEN all 12 meters have status `OK`

#### Scenario: After the demo analysis

- GIVEN the demo analysis completed
- WHEN the client lists meters
- THEN M-109 is `Critical`; M-112 and M-104 are `Alert`; M-106 and the 8 healthy meters are `OK`

### Requirement: Derived daily variation

Daily variation MUST be the latest full day's total kWh versus the meter's baseline daily total, in percent, and MUST NOT depend on a completed analysis.

#### Scenario: Variation without analysis

- GIVEN no analysis has run
- WHEN the client lists meters
- THEN M-109 reports a daily variation of about +109% and a healthy meter reports within ±5%

### Requirement: Meter detail

`GET /api/meters/{id}` MUST return the meter's list fields plus its open findings count. An unknown id MUST return `404` with a JSON error body.

#### Scenario: Known meter

- GIVEN meter M-109 exists
- WHEN the client calls `GET /api/meters/M-109`
- THEN the response is `200` with its fields

#### Scenario: Unknown meter

- GIVEN no meter M-999
- WHEN the client calls `GET /api/meters/M-999`
- THEN the response is `404`

### Requirement: Readings series

`GET /api/meters/{id}/readings` MUST return readings in ascending timestamp order, each with `timestamp`, `consumption_kwh`, `voltage_v`, `current_a`, `power_factor`, `status` and `baseline_kwh` (the hour-of-day baseline for that point). It MUST support optional `from` and `to` (RFC 3339) filters. It MUST return `404` for an unknown meter and `400` for an unparsable or inverted range.

#### Scenario: Full series

- GIVEN meter M-101 with 336 readings
- WHEN the client requests its readings without filters
- THEN 336 points are returned ascending, each with `baseline_kwh`

#### Scenario: Range filter

- GIVEN meter M-101
- WHEN the client requests `from=2026-09-05T00:00:00Z&to=2026-09-05T23:00:00Z`
- THEN exactly the 24 readings of that day are returned

#### Scenario: Unknown meter

- GIVEN no meter M-999
- WHEN the client requests `/api/meters/M-999/readings`
- THEN the response is `404`

#### Scenario: Inverted range

- GIVEN meter M-101
- WHEN `from` is later than `to`
- THEN the response is `400`
