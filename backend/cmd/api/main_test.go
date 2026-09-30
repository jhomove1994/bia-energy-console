package main

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestBootstrapSeedsDemoDataAndIsIdempotent(t *testing.T) {
	env := envOf(map[string]string{"DB_PATH": filepath.Join(t.TempDir(), "e.db"), "DATA_DIR": "../../../data"})
	for i := 0; i < 2; i++ {
		var logs bytes.Buffer
		db, cfg, err := bootstrap(context.Background(), env, slog.New(slog.NewTextHandler(&logs, nil)))
		if err != nil {
			t.Fatalf("bootstrap run %d: %v", i, err)
		}
		db.Close()
		if !strings.Contains(logs.String(), "seed: meters=12 readings=4032 events=4") {
			t.Errorf("run %d: seed log missing:\n%s", i, logs.String())
		}
		if i == 1 && !strings.Contains(logs.String(), "(inserted=0)") {
			t.Errorf("second run must insert nothing:\n%s", logs.String())
		}
		if strings.Contains(logs.String(), cfg.DemoPassword) || strings.Contains(logs.String(), cfg.AuthSecret) {
			t.Errorf("logs leak secrets:\n%s", logs.String())
		}
	}
}

func TestBootstrapFailsWhenDBEmptyAndCSVMissing(t *testing.T) {
	env := envOf(map[string]string{"DB_PATH": filepath.Join(t.TempDir(), "e.db"), "DATA_DIR": t.TempDir()})
	_, _, err := bootstrap(context.Background(), env, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err == nil || !strings.Contains(err.Error(), "readings.csv") {
		t.Fatalf("err = %v, want failure naming readings.csv", err)
	}
}

func TestBootstrapWarnsWhenCSVMissingButDBHasData(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "e.db")
	seedEnv := envOf(map[string]string{"DB_PATH": dbPath, "DATA_DIR": "../../../data"})
	db, _, err := bootstrap(context.Background(), seedEnv, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()

	var logs bytes.Buffer
	env := envOf(map[string]string{"DB_PATH": dbPath, "DATA_DIR": t.TempDir()})
	db, _, err = bootstrap(context.Background(), env, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatalf("bootstrap must continue with existing data: %v", err)
	}
	db.Close()
	if !strings.Contains(logs.String(), "level=WARN") {
		t.Errorf("expected a warning:\n%s", logs.String())
	}
}

func TestBootstrapSkipsSeedWhenDisabled(t *testing.T) {
	var logs bytes.Buffer
	env := envOf(map[string]string{"DB_PATH": filepath.Join(t.TempDir(), "e.db"), "DATA_DIR": t.TempDir(), "SEED_ON_START": "false"})
	db, _, err := bootstrap(context.Background(), env, slog.New(slog.NewTextHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	if strings.Contains(logs.String(), "seed:") {
		t.Errorf("seed must not run:\n%s", logs.String())
	}
}
