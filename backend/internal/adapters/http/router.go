package httpapi

import (
	"log/slog"
	"net/http"

	"energy/internal/app"
	"energy/internal/engine"
)

// Deps are the collaborators the HTTP adapter needs.
type Deps struct {
	DemoUser     string
	DemoPassword string
	Tokens       *Tokens
	Log          *slog.Logger
	Meters       *app.MeterService
	Dashboard    *app.DashboardService
	Analysis     *app.AnalysisService
	Anomalies    *app.AnomalyService
	// Engine supplies the confidence label bands; nil means the defaults.
	Engine *engine.Config
}

type server struct {
	deps Deps
	cfg  engine.Config
}

// NewRouter builds the API handler: recover -> requestLog -> auth -> mux.
func NewRouter(d Deps) http.Handler {
	s := &server{deps: d, cfg: engine.DefaultConfig()}
	if d.Engine != nil {
		s.cfg = *d.Engine
	}
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
	if d.Analysis != nil {
		mux.HandleFunc("POST /api/ai/analyze", s.handleAnalyze)
		mux.HandleFunc("GET /api/ai/analysis/latest", s.handleLatestAnalysis)
		mux.HandleFunc("GET /api/ai/analysis/{id}", s.handleGetAnalysis)
	}
	if d.Anomalies != nil {
		mux.HandleFunc("GET /api/anomalies", s.handleListAnomalies)
		mux.HandleFunc("GET /api/anomalies/{id}", s.handleGetAnomaly)
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "route not found")
	})
	return chain(mux, recoverPanics(d.Log), requestLog(d.Log), authExceptPublic(d.Tokens))
}
