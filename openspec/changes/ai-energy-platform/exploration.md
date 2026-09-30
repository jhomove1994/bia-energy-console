# Exploration: ai-energy-platform

Source: script-based profiling of `docs/readings.csv` and `docs/events.csv` (CSVs never loaded whole into context).

## Dataset facts

- `readings.csv`: 4,032 rows = 12 meters (M-101..M-112) x 336 hourly readings (2026-09-01 00:00 .. 2026-09-14 23:00).
- Columns: `meter_id,timestamp,consumption_kwh,voltage_v,current_a,power_factor,status`.
- No missing values, no timestamp gaps, `status` is `OK` on every row. Gap/missing detection must exist but will produce nothing on this dataset.
- `events.csv`: 4 rows, columns `meter_id,event_timestamp,event_type,description`:
  - M-104 2026-09-11 00:00 `OPERATIONAL_CHANGE` "New production line activated"
  - M-106 2026-09-08 00:00 `SCHEDULED_OUTAGE` "Scheduled maintenance outage for 12 hours"
  - M-109 2026-09-12 14:00 `UNKNOWN` "No operational event reported"
  - M-112 2026-09-13 00:00 `DATA_QUALITY` "Intermittent readings and abnormal electrical jumps"
- Event types carry different explanatory power: `UNKNOWN` must NOT explain an anomaly.

## Healthy-meter noise envelope (8 meters with no events)

- Hourly consumption vs hour-of-day median: max |deviation| 10–17%, zero hours >= 40%.
- Daily kWh varies about ±2%.
- Voltage ~216–225 V, no hour-to-hour jump > 8 V; no current jump > 50%; PF 0.87–0.99.
- Electrical consistency ratio `kWh / (V*A*PF/1000)` per meter: median 1.02–1.12, MAD 0.03–0.05. It is NOT 1.0 in absolute terms, so baseline it per meter.
- Pitfall: point robust-z > 3.5 against a 7-sample hour-of-day baseline fires 6–33 times on healthy meters (tiny MAD). Detection must combine robust-z with a minimum effect size (%) and/or persistence.
- Daily load shape (M-101): ~23 kWh at night, ~36 kWh daytime, so hour-of-day baselines are warranted.

## Target meters

| Meter | Observation | Supports expected result? |
|---|---|---|
| M-109 | From 09-12 14:00, kWh ~50 -> ~110 (+118–130%/h). Daily 1,050 -> 2,194/2,208 (+109%). Current ~195 -> ~410 A, PF 0.94 -> 0.74, V drops ~2–3 V. Ratio 1.07 -> 1.32–1.43. 58 h >= 40% deviation. Event type `UNKNOWN`. | Yes: REAL_ANOMALY, HIGH, top priority |
| M-106 | 09-08 00:00–11:00: kWh -80% (~8–14 kWh), current ~39–61 A, V/PF normal, returns at 12:00. 12 h >= 40%. `SCHEDULED_OUTAGE` 12 h at 09-08 00:00 matches exactly. | Yes: FALSE_POSITIVE, LOW |
| M-104 | From 09-11 00:00, daily 1,168 -> ~1,715 (+47%), current 213 -> 316 A, PF 0.91 -> 0.89, V stable, ratio stable. Persistent step change. `OPERATIONAL_CHANGE` at exactly the step. | Yes: EXPLAINABLE_ANOMALY, MEDIUM |
| M-112 | Daily kWh stable (±1%, no deviation >16%). On 09-13/14 every 3 h: V toggles ~202/~240 V, PF takes quantized values 0.58/0.72/0.98, current jumps 46–218 A, ratio 0.58–4.22. 32 V-jumps > 8 V, 13 current jumps > 50%. `DATA_QUALITY` event at 09-13 00:00. | Yes: DATA_QUALITY, HIGH |

Conclusion: the four expected outcomes are derivable from generic rules without meter-specific logic. All other meters stay within the noise envelope, which matches the summary "4 anomalies detected, 2 require priority attention" (the two HIGH ones: M-109 and M-112).

## Environment

- Local Go 1.21.5 (darwin/arm64), python3, docker are available. Docker builds may pin a newer Go image.
