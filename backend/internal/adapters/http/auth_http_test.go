package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	testUser   = "admin"
	testPass   = "s3cret-pass"
	testSecret = "signing-secret"
)

func newTestRouter(t *testing.T, now time.Time) (http.Handler, *bytes.Buffer) {
	t.Helper()
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	return NewRouter(Deps{
		DemoUser:     testUser,
		DemoPassword: testPass,
		Tokens:       NewTokens(testSecret, time.Hour, fixedClock(now)),
		Log:          log,
	}), &logs
}

func do(h http.Handler, method, path, body, bearer string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct{ Code, Message string } `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error.Code == "" || body.Error.Message == "" {
		t.Fatalf("not a JSON error body: %q (%v)", rec.Body.String(), err)
	}
	return body.Error.Code
}

func TestLoginIssuesTokenForDemoCredentials(t *testing.T) {
	h, _ := newTestRouter(t, t0)
	rec := do(h, "POST", "/api/auth/login", `{"username":"admin","password":"s3cret-pass"}`, "")
	if rec.Code != 200 {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		User      struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Token == "" || !out.ExpiresAt.Equal(t0.Add(time.Hour)) || out.User.Username != "admin" {
		t.Fatalf("login payload = %+v", out)
	}
	if _, err := NewTokens(testSecret, time.Hour, fixedClock(t0)).Verify(out.Token); err != nil {
		t.Fatalf("issued token does not verify: %v", err)
	}
}

func TestLoginRejectsBadCredentialsAndBodies(t *testing.T) {
	h, _ := newTestRouter(t, t0)
	tests := []struct {
		name, body string
		status     int
		code       string
	}{
		{"wrong password", `{"username":"admin","password":"nope"}`, 401, "invalid_credentials"},
		{"wrong user", `{"username":"root","password":"s3cret-pass"}`, 401, "invalid_credentials"},
		{"empty credentials", `{}`, 401, "invalid_credentials"},
		{"malformed json", `{not json`, 400, "invalid_body"},
		{"empty body", ``, 400, "invalid_body"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(h, "POST", "/api/auth/login", tt.body, "")
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if got := errorCode(t, rec); got != tt.code {
				t.Fatalf("code = %q, want %q", got, tt.code)
			}
			if strings.Contains(rec.Body.String(), "token") && !strings.Contains(rec.Body.String(), "error") {
				t.Fatalf("response must not carry a token: %s", rec.Body)
			}
		})
	}
}

func TestHealthIsPublic(t *testing.T) {
	h, _ := newTestRouter(t, t0)
	rec := do(h, "GET", "/api/health", "", "")
	if rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("health = %d %q", rec.Code, rec.Body)
	}
}

func TestProtectedRoutesRequireValidToken(t *testing.T) {
	now := t0.Add(2 * time.Hour)
	h, _ := newTestRouter(t, now)
	valid, _ := NewTokens(testSecret, time.Hour, fixedClock(now)).Issue("admin")
	expired, _ := NewTokens(testSecret, time.Hour, fixedClock(t0)).Issue("admin")
	sig := valid[strings.Index(valid, ".")+1:]
	tampered := "eyJzdWIiOiJyb290IiwiZXhwIjo5OTk5OTk5OTk5fQ." + sig

	for name, bearer := range map[string]string{
		"missing":   "",
		"malformed": "not-a-token",
		"tampered":  tampered,
		"expired":   expired,
	} {
		t.Run(name, func(t *testing.T) {
			rec := do(h, "GET", "/api/meters", "", bearer)
			if rec.Code != 401 || errorCode(t, rec) != "unauthorized" {
				t.Fatalf("status = %d body %s, want 401 unauthorized", rec.Code, rec.Body)
			}
		})
	}

	// Non-bearer scheme is also rejected.
	req := httptest.NewRequest("GET", "/api/meters", nil)
	req.Header.Set("Authorization", "Basic "+valid)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("basic scheme status = %d, want 401", rec.Code)
	}
}

func TestAuthMiddlewareLetsValidTokenThrough(t *testing.T) {
	tk := NewTokens(testSecret, time.Hour, fixedClock(t0))
	token, _ := tk.Issue("admin")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	h := requireAuth(tk)(next)

	if rec := do(h, "GET", "/api/anything", "", token); rec.Code != http.StatusNoContent {
		t.Fatalf("valid token status = %d, want 204", rec.Code)
	}
	if rec := do(h, "GET", "/api/anything", "", ""); rec.Code != 401 {
		t.Fatalf("no token status = %d, want 401", rec.Code)
	}
}

func TestRecoverTurnsPanicIntoJSON500(t *testing.T) {
	var logs bytes.Buffer
	log := slog.New(slog.NewTextHandler(&logs, nil))
	h := recoverPanics(log)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))

	rec := do(h, "GET", "/api/x", "", "")
	if rec.Code != 500 || errorCode(t, rec) != "internal" {
		t.Fatalf("status = %d body %s, want 500 internal", rec.Code, rec.Body)
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Fatal("panic detail must not leak to the client")
	}
	if !strings.Contains(logs.String(), "boom") {
		t.Fatalf("panic must be logged: %s", logs.String())
	}
}

func TestRequestLogDoesNotLeakSecrets(t *testing.T) {
	h, logs := newTestRouter(t, t0)
	rec := do(h, "POST", "/api/auth/login", `{"username":"admin","password":"s3cret-pass"}`, "")
	var out struct{ Token string }
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	do(h, "GET", "/api/meters", "", out.Token)

	got := logs.String()
	if !strings.Contains(got, "/api/auth/login") || !strings.Contains(got, "status=200") {
		t.Fatalf("expected request log lines, got: %q", got)
	}
	for _, secret := range []string{testPass, testSecret, out.Token} {
		if strings.Contains(got, secret) {
			t.Fatalf("log leaks %q: %s", secret, got)
		}
	}
}

func TestUnknownRouteReturnsJSON404(t *testing.T) {
	h, _ := newTestRouter(t, t0)
	token, _ := NewTokens(testSecret, time.Hour, fixedClock(t0)).Issue("admin")
	rec := do(h, "GET", "/api/nope", "", token)
	if rec.Code != 404 || errorCode(t, rec) != "not_found" {
		t.Fatalf("status = %d body %s", rec.Code, rec.Body)
	}
}
