package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"time"
)

const maxBodyBytes = 1 << 20

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      userDTO   `json:"user"`
}

type userDTO struct {
	Username string `json:"username"`
}

// equalConstTime compares two strings without leaking where they differ.
func equalConstTime(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, codeInvalidBody, "request body must be JSON {username, password}")
		return
	}
	// Evaluate both comparisons so timing does not reveal which one failed.
	userOK := equalConstTime(req.Username, s.deps.DemoUser)
	passOK := equalConstTime(req.Password, s.deps.DemoPassword)
	if !userOK || !passOK {
		writeError(w, http.StatusUnauthorized, codeInvalidCredentials, "invalid username or password")
		return
	}
	token, exp := s.deps.Tokens.Issue(req.Username)
	writeJSON(w, http.StatusOK, loginResponse{Token: token, ExpiresAt: exp, User: userDTO{Username: req.Username}})
}
