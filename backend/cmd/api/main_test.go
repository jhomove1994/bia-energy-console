package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestNewHandlerServesSeededDataBehindAuth(t *testing.T) {
	env := envOf(map[string]string{"DB_PATH": filepath.Join(t.TempDir(), "e.db"), "DATA_DIR": "../../../data"})
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, cfg, err := bootstrap(context.Background(), env, log)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	srv := httptest.NewServer(newHandler(db, cfg, log))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/meters")
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated GET /api/meters = %v, %v; want 401", resp, err)
	}

	body := strings.NewReader(`{"username":"admin","password":"admin123"}`)
	resp, err = http.Post(srv.URL+"/api/auth/login", "application/json", body)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("login = %v, %v", resp, err)
	}
	var login struct{ Token string }
	if err := json.NewDecoder(resp.Body).Decode(&login); err != nil || login.Token == "" {
		t.Fatalf("login body: %v", err)
	}

	req, _ := http.NewRequest("GET", srv.URL+"/api/meters", nil)
	req.Header.Set("Authorization", "Bearer "+login.Token)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/meters = %v, %v", resp, err)
	}
	var list struct {
		Items []struct {
			MeterID string `json:"meter_id"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil || len(list.Items) != 12 {
		t.Fatalf("meters = %d, err %v; want 12", len(list.Items), err)
	}
}

func TestServeStopsGracefullyOnContextCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	done := make(chan error, 1)
	go func() {
		done <- serve(ctx, &http.Server{Handler: handler}, ln, slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()

	resp, err := http.Get("http://" + ln.Addr().String())
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("GET while serving = %v, %v", resp, err)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve returned %v after cancel, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not stop after context cancel")
	}
	if _, err := http.Get("http://" + ln.Addr().String()); err == nil {
		t.Fatal("listener still accepting after shutdown")
	}
}
