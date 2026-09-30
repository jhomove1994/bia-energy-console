package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"energy/internal/app"
	"energy/internal/domain"
)

// evidenceRecord is the stored shape of domain.Evidence.
type evidenceRecord struct {
	Metric   string   `json:"metric"`
	Label    string   `json:"label"`
	Baseline *float64 `json:"baseline"`
	Observed float64  `json:"observed"`
	DeltaPct *float64 `json:"delta_pct"`
	Unit     string   `json:"unit"`
	Detail   string   `json:"detail"`
	EventID  *int64   `json:"event_id"`
}

func encodeEvidence(ev []domain.Evidence) (string, error) {
	recs := make([]evidenceRecord, len(ev))
	for i, e := range ev {
		recs[i] = evidenceRecord(e)
	}
	raw, err := json.Marshal(recs)
	if err != nil {
		return "", fmt.Errorf("encode evidence: %w", err)
	}
	return string(raw), nil
}

func decodeEvidence(raw string) ([]domain.Evidence, error) {
	var recs []evidenceRecord
	if err := json.Unmarshal([]byte(raw), &recs); err != nil {
		return nil, fmt.Errorf("decode evidence: %w", err)
	}
	out := make([]domain.Evidence, len(recs))
	for i, r := range recs {
		out[i] = domain.Evidence(r)
	}
	return out, nil
}

// insertAnomalies writes anomalies inside the caller's transaction.
func insertAnomalies(ctx context.Context, tx *sql.Tx, as []domain.Anomaly) error {
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO anomalies
		(id, analysis_id, meter_id, detected_at, window_start, window_end, kind, type, severity, severity_score,
		 confidence, priority_score, rank, action_code, reason, recommended_action, related_event_id, evidence_json, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare anomalies insert: %w", err)
	}
	defer stmt.Close()
	for _, a := range as {
		ev, err := encodeEvidence(a.Evidence)
		if err != nil {
			return err
		}
		var eventID any
		if a.RelatedEvent != nil {
			eventID = a.RelatedEvent.ID
		}
		status := a.Status
		if status == "" {
			status = "open"
		}
		if _, err := stmt.ExecContext(ctx, a.ID, a.AnalysisID, a.MeterID, formatTime(a.DetectedAt), formatTime(a.WindowStart),
			formatTime(a.WindowEnd), string(a.Kind), string(a.Type), string(a.Severity), a.SeverityScore, a.Confidence,
			a.PriorityScore, a.Rank, string(a.Action), a.Reason, a.RecommendedAction, eventID, ev, status); err != nil {
			return fmt.Errorf("insert anomaly %s: %w", a.ID, err)
		}
	}
	return nil
}

// AnomalyStore reads persisted anomalies.
type AnomalyStore struct{ db *DB }

var _ app.AnomalyRepository = (*AnomalyStore)(nil)

// NewAnomalyStore returns an AnomalyStore over db.
func NewAnomalyStore(db *DB) *AnomalyStore { return &AnomalyStore{db: db} }

const anomalySelect = `SELECT a.id, a.analysis_id, a.meter_id, a.detected_at, a.window_start, a.window_end, a.kind, a.type,
	a.severity, a.severity_score, a.confidence, a.priority_score, a.rank, a.action_code, a.reason, a.recommended_action,
	a.evidence_json, a.status, e.id, e.meter_id, e.ts, e.type, e.description
	FROM anomalies a LEFT JOIN events e ON e.id = a.related_event_id`

// List returns the anomalies of one analysis ordered by rank.
func (s *AnomalyStore) List(ctx context.Context, analysisID string, f app.AnomalyFilter) ([]domain.Anomaly, error) {
	where, args := []string{"a.analysis_id = ?"}, []any{analysisID}
	if f.MeterID != "" {
		where, args = append(where, "a.meter_id = ?"), append(args, f.MeterID)
	}
	if f.Type != "" {
		where, args = append(where, "a.type = ?"), append(args, string(f.Type))
	}
	if f.Severity != "" {
		where, args = append(where, "a.severity = ?"), append(args, string(f.Severity))
	}
	rows, err := s.db.QueryContext(ctx, anomalySelect+" WHERE "+strings.Join(where, " AND ")+" ORDER BY a.rank, a.id", args...)
	if err != nil {
		return nil, fmt.Errorf("list anomalies of %s: %w", analysisID, err)
	}
	defer rows.Close()
	var out []domain.Anomaly
	for rows.Next() {
		a, err := scanAnomaly(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Get returns one anomaly, or ErrNotFound.
func (s *AnomalyStore) Get(ctx context.Context, id string) (domain.Anomaly, error) {
	a, err := scanAnomaly(s.db.QueryRowContext(ctx, anomalySelect+" WHERE a.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Anomaly{}, fmt.Errorf("anomaly %s %w", id, app.ErrNotFound)
	}
	return a, err
}

func scanAnomaly(row rowScanner) (domain.Anomaly, error) {
	var a domain.Anomaly
	var detected, wStart, wEnd, kind, typ, sev, action, evidence string
	var evID sql.NullInt64
	var evMeter, evTS, evType, evDesc sql.NullString
	if err := row.Scan(&a.ID, &a.AnalysisID, &a.MeterID, &detected, &wStart, &wEnd, &kind, &typ, &sev, &a.SeverityScore,
		&a.Confidence, &a.PriorityScore, &a.Rank, &action, &a.Reason, &a.RecommendedAction, &evidence, &a.Status,
		&evID, &evMeter, &evTS, &evType, &evDesc); err != nil {
		return domain.Anomaly{}, err
	}
	a.Kind, a.Type, a.Severity, a.Action = domain.AnomalyKind(kind), domain.AnomalyType(typ), domain.Severity(sev), domain.ActionCode(action)
	var err error
	if a.DetectedAt, err = parseTime(detected); err != nil {
		return domain.Anomaly{}, err
	}
	if a.WindowStart, err = parseTime(wStart); err != nil {
		return domain.Anomaly{}, err
	}
	if a.WindowEnd, err = parseTime(wEnd); err != nil {
		return domain.Anomaly{}, err
	}
	if a.Evidence, err = decodeEvidence(evidence); err != nil {
		return domain.Anomaly{}, err
	}
	if evID.Valid {
		ev := domain.Event{ID: evID.Int64, MeterID: evMeter.String, Type: domain.EventType(evType.String), Description: evDesc.String}
		if ev.TS, err = parseTime(evTS.String); err != nil {
			return domain.Anomaly{}, err
		}
		a.RelatedEvent = &ev
	}
	return a, nil
}
