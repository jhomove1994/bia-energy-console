package platform

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" || cfg.DBPath != "./energy.db" || cfg.DataDir != "../data" {
		t.Errorf("unexpected paths/port: %+v", cfg)
	}
	if !cfg.SeedOnStart {
		t.Error("SeedOnStart default must be true")
	}
	if cfg.DemoUser != "admin" || cfg.DemoPassword != "admin123" {
		t.Errorf("unexpected demo credentials: %q/%q", cfg.DemoUser, cfg.DemoPassword)
	}
	if cfg.AuthSecret == "" {
		t.Error("AuthSecret must default to a dev constant")
	}
	if cfg.TokenTTL != 12*time.Hour {
		t.Errorf("TokenTTL = %v, want 12h", cfg.TokenTTL)
	}
	if cfg.StageDelay != 0 {
		t.Errorf("StageDelay = %v, want 0", cfg.StageDelay)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"PORT": "9090", "DB_PATH": "/tmp/x.db", "DATA_DIR": "/data",
		"SEED_ON_START": "false", "DEMO_USER": "bob", "DEMO_PASSWORD": "pw",
		"AUTH_SECRET": "s3", "TOKEN_TTL": "30m", "ANALYSIS_STAGE_DELAY_MS": "300",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" || cfg.DBPath != "/tmp/x.db" || cfg.DataDir != "/data" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	if cfg.SeedOnStart || cfg.DemoUser != "bob" || cfg.DemoPassword != "pw" || cfg.AuthSecret != "s3" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
	if cfg.TokenTTL != 30*time.Minute || cfg.StageDelay != 300*time.Millisecond {
		t.Errorf("duration overrides not applied: %v %v", cfg.TokenTTL, cfg.StageDelay)
	}
}

func TestLoadInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"bad duration", map[string]string{"TOKEN_TTL": "soon"}, "TOKEN_TTL"},
		{"bad delay", map[string]string{"ANALYSIS_STAGE_DELAY_MS": "abc"}, "ANALYSIS_STAGE_DELAY_MS"},
		{"negative delay", map[string]string{"ANALYSIS_STAGE_DELAY_MS": "-5"}, "ANALYSIS_STAGE_DELAY_MS"},
		{"bad bool", map[string]string{"SEED_ON_START": "maybe"}, "SEED_ON_START"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(env(tt.env))
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want mention of %s", err, tt.want)
			}
		})
	}
}

func TestConfigStringRedactsSecrets(t *testing.T) {
	cfg, _ := Load(env(map[string]string{"DEMO_PASSWORD": "hunter2", "AUTH_SECRET": "topsecret"}))
	s := cfg.String()
	if strings.Contains(s, "hunter2") || strings.Contains(s, "topsecret") {
		t.Errorf("String leaks secrets: %s", s)
	}
	if !strings.Contains(s, "port=8080") {
		t.Errorf("String missing port: %s", s)
	}
}
