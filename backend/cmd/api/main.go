// Command api boots the energy platform backend: config, SQLite, migration and seed.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"energy/internal/adapters/csvload"
	"energy/internal/adapters/sqlite"
	"energy/internal/platform"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	db, _, err := bootstrap(context.Background(), os.Getenv, log)
	if err != nil {
		log.Error("startup failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	log.Info("bootstrap complete; HTTP server is not wired yet")
}

// bootstrap loads config, opens and migrates the database and seeds it from
// CSV when enabled. A missing CSV is fatal on an empty database and only a
// warning when the database already holds data.
func bootstrap(ctx context.Context, getenv func(string) string, log *slog.Logger) (*sqlite.DB, platform.Config, error) {
	cfg, err := platform.Load(getenv)
	if err != nil {
		return nil, cfg, fmt.Errorf("load config: %w", err)
	}
	log.Info("config loaded", "config", cfg.String())
	if cfg.AuthSecret == platform.DevAuthSecret {
		log.Warn("AUTH_SECRET is the built-in development default; set it outside demos")
	}

	db, err := sqlite.Open(cfg.DBPath)
	if err != nil {
		return nil, cfg, err
	}
	fail := func(err error) (*sqlite.DB, platform.Config, error) {
		db.Close()
		return nil, cfg, err
	}
	if err := sqlite.Migrate(ctx, db); err != nil {
		return fail(err)
	}

	if cfg.SeedOnStart {
		if _, err := csvload.Seed(ctx, db, cfg.DataDir, log); err != nil {
			if !errors.Is(err, csvload.ErrDataMissing) {
				return fail(err)
			}
			var meters int
			if qerr := db.QueryRowContext(ctx, "SELECT count(*) FROM meters").Scan(&meters); qerr != nil {
				return fail(qerr)
			}
			if meters == 0 {
				return fail(fmt.Errorf("database is empty and seed data is unavailable: %w", err))
			}
			log.Warn("seed data unavailable; continuing with existing database", "error", err.Error(), "meters", meters)
		}
	}
	return db, cfg, nil
}
