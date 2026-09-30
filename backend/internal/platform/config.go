// Package platform holds process-level concerns: environment configuration.
package platform

import (
	"fmt"
	"strconv"
	"time"
)

// Config is the API runtime configuration, read from the environment.
type Config struct {
	Port         string
	DBPath       string
	DataDir      string
	SeedOnStart  bool
	DemoUser     string
	DemoPassword string
	AuthSecret   string
	TokenTTL     time.Duration
	StageDelay   time.Duration
}

// DevAuthSecret is the insecure default used when AUTH_SECRET is unset.
const DevAuthSecret = "change-me-demo"

// Load builds a Config from getenv, applying documented defaults.
func Load(getenv func(string) string) (Config, error) {
	str := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	cfg := Config{
		Port:         str("PORT", "8080"),
		DBPath:       str("DB_PATH", "./energy.db"),
		DataDir:      str("DATA_DIR", "../data"),
		DemoUser:     str("DEMO_USER", "admin"),
		DemoPassword: str("DEMO_PASSWORD", "admin123"),
		AuthSecret:   str("AUTH_SECRET", DevAuthSecret),
	}

	seed := true
	if v := getenv("SEED_ON_START"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("SEED_ON_START: invalid boolean %q", v)
		}
		seed = b
	}
	cfg.SeedOnStart = seed

	ttl := 12 * time.Hour
	if v := getenv("TOKEN_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("TOKEN_TTL: invalid duration %q", v)
		}
		ttl = d
	}
	cfg.TokenTTL = ttl

	if v := getenv("ANALYSIS_STAGE_DELAY_MS"); v != "" {
		ms, err := strconv.Atoi(v)
		if err != nil || ms < 0 {
			return Config{}, fmt.Errorf("ANALYSIS_STAGE_DELAY_MS: invalid non-negative integer %q", v)
		}
		cfg.StageDelay = time.Duration(ms) * time.Millisecond
	}
	return cfg, nil
}

// String returns a log-safe summary; the password and secret are redacted.
func (c Config) String() string {
	return fmt.Sprintf("port=%s db_path=%s data_dir=%s seed_on_start=%t demo_user=%s demo_password=*** auth_secret=*** token_ttl=%s stage_delay=%s",
		c.Port, c.DBPath, c.DataDir, c.SeedOnStart, c.DemoUser, c.TokenTTL, c.StageDelay)
}
