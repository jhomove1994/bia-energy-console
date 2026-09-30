package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type middleware func(http.Handler) http.Handler

// chain applies mws so that the first one is the outermost.
func chain(h http.Handler, mws ...middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// recoverPanics turns a handler panic into a logged JSON 500.
func recoverPanics(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic in handler", "method", r.Method, "path", r.URL.Path, "panic", rec)
					writeError(w, http.StatusInternalServerError, codeInternal, "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// requestLog logs method, path, status and duration. It never logs headers or
// bodies, so credentials and tokens stay out of the logs.
func requestLog(log *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("request", "method", r.Method, "path", r.URL.Path, "status", rec.status, "duration", time.Since(start))
		})
	}
}

// requireAuth rejects requests without a valid bearer token.
func requireAuth(tokens *Tokens) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || token == "" {
				writeError(w, http.StatusUnauthorized, codeUnauthorized, "missing or malformed bearer token")
				return
			}
			if _, err := tokens.Verify(token); err != nil {
				writeError(w, http.StatusUnauthorized, codeUnauthorized, "invalid or expired token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// publicRoutes bypass authentication.
var publicRoutes = map[string]bool{
	"GET /api/health":      true,
	"POST /api/auth/login": true,
}

// authExceptPublic applies auth to every route that is not public.
func authExceptPublic(tokens *Tokens) middleware {
	protect := requireAuth(tokens)
	return func(next http.Handler) http.Handler {
		protected := protect(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicRoutes[r.Method+" "+r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}
