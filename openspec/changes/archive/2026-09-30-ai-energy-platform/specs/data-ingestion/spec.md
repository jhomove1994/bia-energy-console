# Data Ingestion Specification

## Purpose

Load hourly meter readings and operational events from CSV into SQLite at startup, safely and repeatably, without altering the source data.

## Requirements

### Requirement: Startup seed from CSV

The system MUST load `readings.csv` and `events.csv` from a configurable data directory into SQLite during startup, before the HTTP server accepts requests.

#### Scenario: Fresh database is seeded

- GIVEN an empty SQLite database and the demo CSV files
- WHEN the API starts
- THEN 12 meters, 4,032 readings and 4 events are stored
- AND the API accepts requests only after seeding completes

#### Scenario: Data file is missing

- GIVEN the configured data directory has no `readings.csv`
- WHEN the API starts
- THEN startup fails with a non-zero exit and a message naming the missing file

### Requirement: Idempotent seed

Seeding MUST be idempotent. A reading is identified by (meter_id, timestamp); re-seeding MUST NOT create duplicates or overwrite existing rows.

#### Scenario: Restart with existing data

- GIVEN a database already seeded with the demo CSV files
- WHEN the API restarts
- THEN row counts stay 12 meters, 4,032 readings and 4 events

#### Scenario: Partially seeded database

- GIVEN a database containing only some of the CSV readings
- WHEN the API starts
- THEN the missing readings are added and existing rows are unchanged

### Requirement: Row validation

The system MUST skip malformed rows (wrong column count, unparsable timestamp or number) and log the file, line and reason. Valid rows MUST still load.

#### Scenario: Malformed row in the file

- GIVEN a readings file with one row whose `voltage_v` is not numeric
- WHEN the API seeds
- THEN that row is skipped and logged with its line number
- AND all other rows are stored

### Requirement: Event type normalization

Event types MUST be stored as one of the known types (`SCHEDULED_OUTAGE`, `OPERATIONAL_CHANGE`, `DATA_QUALITY`, `UNKNOWN`). Any unrecognized type MUST be stored as `UNKNOWN`, preserving the original description.

#### Scenario: Unrecognized event type

- GIVEN an events row with type `FIRMWARE_UPDATE`
- WHEN the API seeds
- THEN the event is stored with type `UNKNOWN` and its original description

### Requirement: Reading fidelity and cadence reporting

The system MUST store `status` exactly as given and MUST NOT interpolate, fill or discard readings. Hourly cadence gaps per meter MUST remain detectable from stored data, and the seed SHOULD log a per-meter gap count.

#### Scenario: Gap in a series

- GIVEN a meter whose series lacks the 05:00 reading on one day
- WHEN the API seeds
- THEN no 05:00 row is created
- AND the seed log reports one gap for that meter

#### Scenario: Non-OK status preserved

- GIVEN a reading with status `SENSOR_FAULT`
- WHEN the API seeds
- THEN the stored reading has status `SENSOR_FAULT`

#### Scenario: Demo dataset is clean

- GIVEN the demo CSV files
- WHEN the API seeds
- THEN the cadence report shows zero gaps and zero non-OK statuses

### Requirement: Meter registry without invented attributes

Meters MUST be derived from distinct `meter_id` values in the readings. The system MUST NOT store or synthesize a meter name or location.

#### Scenario: Meter identity only

- GIVEN the seeded database
- WHEN a meter record is inspected
- THEN it contains the `meter_id` and creation time and no name or location field
