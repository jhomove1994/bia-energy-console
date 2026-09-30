package httpapi

import (
	"log/slog"
	"net/http"
)

// Deps are the collaborators the HTTP adapter needs.
type Deps struct {
	DemoUser     string
	DemoPassword string
	Tokens       *Tokens
	Log          *slog.Logger
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
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, codeNotFound, "route not found")
	})
	return chain(mux, recoverPanics(d.Log), requestLog(d.Log), authExceptPublic(d.Tokens))
}
