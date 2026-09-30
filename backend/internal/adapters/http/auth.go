// Package httpapi implements the HTTP adapter: routing, middleware, stateless
// token authentication and the JSON handlers.
package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrInvalidToken reports a malformed, tampered or wrongly signed token.
	ErrInvalidToken = errors.New("invalid token")
	// ErrExpiredToken reports a correctly signed token past its expiry.
	ErrExpiredToken = errors.New("token expired")
)

// Claims is the signed token payload.
type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"` // unix seconds
}

// Tokens issues and verifies stateless HMAC-signed tokens of the form
// base64url(json claims) "." base64url(HMAC-SHA256(secret, payload)).
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokens returns a Tokens with the given secret and lifetime. now supplies
// the current time.
func NewTokens(secret string, ttl time.Duration, now func() time.Time) *Tokens {
	return &Tokens{secret: []byte(secret), ttl: ttl, now: now}
}

var b64 = base64.RawURLEncoding

func (t *Tokens) sign(payload string) []byte {
	m := hmac.New(sha256.New, t.secret)
	m.Write([]byte(payload))
	return m.Sum(nil)
}

// Issue returns a token for sub and its expiry time.
func (t *Tokens) Issue(sub string) (string, time.Time) {
	exp := t.now().Add(t.ttl)
	raw, _ := json.Marshal(Claims{Sub: sub, Exp: exp.Unix()})
	payload := b64.EncodeToString(raw)
	return payload + "." + b64.EncodeToString(t.sign(payload)), exp
}

// Verify checks the signature in constant time and the expiry.
func (t *Tokens) Verify(token string) (Claims, error) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || payload == "" || sig == "" || strings.Contains(sig, ".") {
		return Claims{}, ErrInvalidToken
	}
	got, err := b64.DecodeString(sig)
	if err != nil || subtle.ConstantTimeCompare(got, t.sign(payload)) != 1 {
		return Claims{}, ErrInvalidToken
	}
	raw, err := b64.DecodeString(payload)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !t.now().Before(time.Unix(c.Exp, 0)) {
		return Claims{}, ErrExpiredToken
	}
	return c, nil
}
