-- Embedded schema, applied when PRAGMA user_version < 1.
CREATE TABLE IF NOT EXISTS meters (
  id         INTEGER PRIMARY KEY,
  meter_id   TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS readings (
  id              INTEGER PRIMARY KEY,
  meter_id        TEXT NOT NULL REFERENCES meters(meter_id),
  ts              TEXT NOT NULL,
  consumption_kwh REAL NOT NULL,
  voltage_v       REAL NOT NULL,
  current_a       REAL NOT NULL,
  power_factor    REAL NOT NULL,
  status          TEXT NOT NULL,
  UNIQUE (meter_id, ts)
);
CREATE TABLE IF NOT EXISTS events (
  id          INTEGER PRIMARY KEY,
  meter_id    TEXT NOT NULL REFERENCES meters(meter_id),
  ts          TEXT NOT NULL,
  type        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  UNIQUE (meter_id, ts, type)
);
CREATE TABLE IF NOT EXISTS analyses (
  id           TEXT PRIMARY KEY,
  status       TEXT NOT NULL CHECK (status IN ('pending','running','done','failed')),
  stage        TEXT NOT NULL DEFAULT '',
  stage_index  INTEGER NOT NULL DEFAULT 0,
  progress     INTEGER NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
  created_at   TEXT NOT NULL,
  started_at   TEXT,
  finished_at  TEXT,
  error        TEXT NOT NULL DEFAULT '',
  summary_json TEXT
);
CREATE INDEX IF NOT EXISTS idx_analyses_status  ON analyses(status);
CREATE INDEX IF NOT EXISTS idx_analyses_created ON analyses(created_at DESC);
CREATE TABLE IF NOT EXISTS anomalies (
  id                 TEXT PRIMARY KEY,
  analysis_id        TEXT NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
  meter_id           TEXT NOT NULL REFERENCES meters(meter_id),
  detected_at        TEXT NOT NULL,
  window_start       TEXT NOT NULL,
  window_end         TEXT NOT NULL,
  kind               TEXT NOT NULL,
  type               TEXT NOT NULL,
  severity           TEXT NOT NULL,
  severity_score     REAL NOT NULL,
  confidence         REAL NOT NULL,
  priority_score     REAL NOT NULL,
  rank               INTEGER NOT NULL,
  action_code        TEXT NOT NULL,
  reason             TEXT NOT NULL,
  recommended_action TEXT NOT NULL,
  related_event_id   INTEGER REFERENCES events(id),
  evidence_json      TEXT NOT NULL,
  status             TEXT NOT NULL DEFAULT 'open'
);
CREATE INDEX IF NOT EXISTS idx_anomalies_analysis_rank ON anomalies(analysis_id, rank);
CREATE INDEX IF NOT EXISTS idx_anomalies_meter         ON anomalies(meter_id);
