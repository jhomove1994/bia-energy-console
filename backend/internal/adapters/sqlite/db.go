// Package sqlite implements the persistence adapters on top of SQLite.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "modernc.org/sqlite" // pure-Go driver, registers "sqlite"
)

// DB wraps *sql.DB configured for a single writer with WAL.
type DB struct {
	*sql.DB
}

// timeLayout is the storage format of every timestamp column (RFC3339, UTC).
const timeLayout = "2006-01-02T15:04:05Z"

// Open opens (creating if needed) the SQLite file at path.
func Open(path string) (*DB, error) {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	dsn := "file:" + path + "?" + q.Encode()

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite %s: %w", path, err)
	}
	return &DB{sqlDB}, nil
}

// Writer performs seed writes inside a single transaction.
type Writer struct {
	tx *sql.Tx
}

// WriteTx runs fn inside one transaction; it commits when fn returns nil.
func (db *DB) WriteTx(ctx context.Context, fn func(w *Writer) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := fn(&Writer{tx: tx}); err != nil {
		return err
	}
	return tx.Commit()
}

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }
