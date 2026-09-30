package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"energy/internal/app"
	"energy/internal/domain"
)

// interruptedMessage is stored on analyses cut short by a process restart.
const interruptedMessage = "interrupted by restart"

// AnalysisStore persists analysis runs.
type AnalysisStore struct {
	db  *DB
	now func() time.Time
}

var _ app.AnalysisRepository = (*AnalysisStore)(nil)

// NewAnalysisStore returns an AnalysisStore over db.
func NewAnalysisStore(db *DB) *AnalysisStore { return &AnalysisStore{db: db, now: time.Now} }

// Create inserts a new analysis row.
func (s *AnalysisStore) Create(ctx context.Context, a domain.Analysis) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO analyses (id, status, stage, stage_index, progress, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		a.ID, string(a.Status), string(a.Stage), a.StageIndex, a.Progress, formatTime(a.CreatedAt))
	if err != nil {
		return fmt.Errorf("create analysis %s: %w", a.ID, err)
	}
	return nil
}

// UpdateProgress records the status, stage and progress. The first update to
// running also stamps started_at.
func (s *AnalysisStore) UpdateProgress(ctx context.Context, id string, st domain.AnalysisStatus, stage domain.Stage, idx, progress int) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET status = ?, stage = ?, stage_index = ?, progress = ?,
		   started_at = CASE WHEN ? = 'running' AND started_at IS NULL THEN ? ELSE started_at END
		 WHERE id = ?`,
		string(st), string(stage), idx, progress, string(st), formatTime(s.now()), id)
	if err != nil {
		return fmt.Errorf("update analysis %s: %w", id, err)
	}
	return requireRow(res, "analysis "+id)
}

// Complete marks the analysis done with progress 100 and inserts its anomalies
// in one transaction: any failure leaves the analysis and the anomalies as
// they were.
func (s *AnalysisStore) Complete(ctx context.Context, id string, finished time.Time, sum domain.Summary, as []domain.Anomaly) error {
	raw, err := json.Marshal(sum)
	if err != nil {
		return fmt.Errorf("encode summary: %w", err)
	}
	return s.db.WriteTx(ctx, func(w *Writer) error {
		res, err := w.tx.ExecContext(ctx,
			`UPDATE analyses SET status = 'done', progress = 100, finished_at = ?, error = '', summary_json = ? WHERE id = ?`,
			formatTime(finished), string(raw), id)
		if err != nil {
			return fmt.Errorf("complete analysis %s: %w", id, err)
		}
		if err := requireRow(res, "analysis "+id); err != nil {
			return err
		}
		return insertAnomalies(ctx, w.tx, as)
	})
}

// Fail marks the analysis failed with msg.
func (s *AnalysisStore) Fail(ctx context.Context, id string, finished time.Time, msg string) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET status = 'failed', finished_at = ?, error = ? WHERE id = ?`,
		formatTime(finished), msg, id)
	if err != nil {
		return fmt.Errorf("fail analysis %s: %w", id, err)
	}
	return requireRow(res, "analysis "+id)
}

const analysisColumns = `id, status, stage, stage_index, progress, created_at, started_at, finished_at, error, summary_json`

// Get returns the analysis, or ErrNotFound.
func (s *AnalysisStore) Get(ctx context.Context, id string) (domain.Analysis, error) {
	a, err := scanAnalysis(s.db.QueryRowContext(ctx, `SELECT `+analysisColumns+` FROM analyses WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Analysis{}, fmt.Errorf("analysis %s %w", id, app.ErrNotFound)
	}
	return a, err
}

// Latest returns the most recently created analysis in any state, or nil.
func (s *AnalysisStore) Latest(ctx context.Context) (*domain.Analysis, error) {
	return s.one(ctx, `SELECT `+analysisColumns+` FROM analyses ORDER BY created_at DESC, rowid DESC LIMIT 1`)
}

// LatestDone returns the most recent done analysis, or nil.
func (s *AnalysisStore) LatestDone(ctx context.Context) (*domain.Analysis, error) {
	return s.one(ctx, `SELECT `+analysisColumns+` FROM analyses WHERE status = 'done' ORDER BY created_at DESC, rowid DESC LIMIT 1`)
}

func (s *AnalysisStore) one(ctx context.Context, query string) (*domain.Analysis, error) {
	a, err := scanAnalysis(s.db.QueryRowContext(ctx, query))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// FailInterrupted marks every pending or running analysis failed and returns
// how many were changed.
func (s *AnalysisStore) FailInterrupted(ctx context.Context, finished time.Time) (int, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE analyses SET status = 'failed', error = ?, finished_at = ? WHERE status IN ('pending', 'running')`,
		interruptedMessage, formatTime(finished))
	if err != nil {
		return 0, fmt.Errorf("fail interrupted analyses: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scanAnalysis(row rowScanner) (domain.Analysis, error) {
	var a domain.Analysis
	var status, stage, created string
	var started, finished, summary sql.NullString
	if err := row.Scan(&a.ID, &status, &stage, &a.StageIndex, &a.Progress, &created, &started, &finished, &a.Error, &summary); err != nil {
		return domain.Analysis{}, err
	}
	a.Status, a.Stage = domain.AnalysisStatus(status), domain.Stage(stage)
	var err error
	if a.CreatedAt, err = parseTime(created); err != nil {
		return domain.Analysis{}, err
	}
	if a.StartedAt, err = parseNullTime(started); err != nil {
		return domain.Analysis{}, err
	}
	if a.FinishedAt, err = parseNullTime(finished); err != nil {
		return domain.Analysis{}, err
	}
	if summary.Valid {
		var sum domain.Summary
		if err := json.Unmarshal([]byte(summary.String), &sum); err != nil {
			return domain.Analysis{}, fmt.Errorf("decode summary of %s: %w", a.ID, err)
		}
		a.Summary = &sum
	}
	return a, nil
}

func parseNullTime(s sql.NullString) (*time.Time, error) {
	if !s.Valid {
		return nil, nil
	}
	t, err := parseTime(s.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func requireRow(res sql.Result, what string) error {
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("%s %w", what, app.ErrNotFound)
	}
	return nil
}
