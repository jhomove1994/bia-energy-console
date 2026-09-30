package httpapi

import (
	"log/slog"
	"net/http"

	"energy/internal/app"
)

// Deps are the collaborators the HTTP adapter needs.
type Deps struct {
	DemoUser     string
	DemoPassword string
	Tokens       *Tokens
	Log          *slog.Logger
	Meters       *app.MeterService
	Dashboard    *app.DashboardService
}

type server struct {
	deps Deps
}

// NewRouter builds the API handler: recover -> requestLog -> auth -> mux.
func NewRouter(d Deps) http.Handler {
	s := &server{deps: d}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	if d.Meters != nil {
		mux.HandleFunc("GET /api/meters", s.handleListMeters)
		mux.HandleFunc("GET /api/meters/{id}", s.handleMeterDetail)
		mux.HandleFunc("GET /api/meters/{id}/readings", s.handleMeterReadings)
		mux.HandleFunc("GET /api/meters/{id}/events", s.handleMeterEvents)
	}
	if d.Dashboard != nil {
		mux.HandleFunc("GET /api/dashboard/summary", s.handleDashboardSummary)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "route not found")
	})
	return chain(mux, recoverPanics(d.Log), requestLog(d.Log), authExceptPublic(d.Tokens))
}
