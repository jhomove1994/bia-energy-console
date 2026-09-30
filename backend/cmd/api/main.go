// Command api boots the energy platform backend: config, SQLite, migration and seed.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"energy/internal/adapters/csvload"
	httpapi "energy/internal/adapters/http"
	"energy/internal/adapters/sqlite"
	"energy/internal/app"
	"energy/internal/engine"
	"energy/internal/platform"
)

const shutdownTimeout = 10 * time.Second

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Getenv, log); err != nil {
		log.Error("startup failed", "error", err)
		os.Exit(1)
	}
}

// run bootstraps the platform and serves HTTP until ctx is cancelled. The
// listener only opens after the seed has completed.
func run(ctx context.Context, getenv func(string) string, log *slog.Logger) error {
	db, cfg, err := bootstrap(ctx, getenv, log)
	if err != nil {
		return err
	}
	defer db.Close()

	ln, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen on port %s: %w", cfg.Port, err)
	}
	srv := &http.Server{Handler: newHandler(db, cfg, log), ReadHeaderTimeout: 10 * time.Second}
	log.Info("http server listening", "addr", ln.Addr().String())
	return serve(ctx, srv, ln, log)
}

// newHandler wires the repositories, services and HTTP adapter.
func newHandler(db *sqlite.DB, cfg platform.Config, log *slog.Logger) http.Handler {
	meters, readings, events := sqlite.NewMeterStore(db), sqlite.NewReadingStore(db), sqlite.NewEventStore(db)
	return httpapi.NewRouter(httpapi.Deps{
		DemoUser:     cfg.DemoUser,
		DemoPassword: cfg.DemoPassword,
		Tokens:       httpapi.NewTokens(cfg.AuthSecret, cfg.TokenTTL, time.Now),
		Log:          log,
		Meters:       app.NewMeterService(meters, readings, events, app.NoAnalysis{}, engine.DefaultConfig()),
		Dashboard:    app.NewDashboardService(meters, readings, app.NoLatestDone{}),
	})
}

// serve runs srv on ln until it fails or ctx is cancelled, then shuts down
// gracefully, letting in-flight requests finish.
func serve(ctx context.Context, srv *http.Server, ln net.Listener, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
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
