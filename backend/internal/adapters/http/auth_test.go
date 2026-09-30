package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

func TestTokenRoundTrip(t *testing.T) {
	tk := NewTokens("secret-a", time.Hour, fixedClock(t0))
	token, exp := tk.Issue("admin")
	if !exp.Equal(t0.Add(time.Hour)) {
		t.Fatalf("exp = %v, want %v", exp, t0.Add(time.Hour))
	}
	claims, err := tk.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.Sub != "admin" || claims.Exp != exp.Unix() {
		t.Fatalf("claims = %+v", claims)
	}
}

func TestTokenRejectsTamperedPayload(t *testing.T) {
	tk := NewTokens("secret-a", time.Hour, fixedClock(t0))
	token, _ := tk.Issue("admin")
	sig := token[strings.Index(token, ".")+1:]

	forged, _ := json.Marshal(map[string]any{"sub": "root", "exp": t0.Add(48 * time.Hour).Unix()})
	tampered := base64.RawURLEncoding.EncodeToString(forged) + "." + sig
	if _, err := tk.Verify(tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("tampered payload: err = %v, want ErrInvalidToken", err)
	}

	flipped := token[:len(token)-2] + "AA"
	if flipped == token {
		flipped = token[:len(token)-2] + "BB"
	}
	if _, err := tk.Verify(flipped); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("tampered signature: err = %v, want ErrInvalidToken", err)
	}
}

func TestTokenRejectsMalformed(t *testing.T) {
	tk := NewTokens("secret-a", time.Hour, fixedClock(t0))
	for _, bad := range []string{"", "abc", "a.b.c", "!!!.???", "."} {
		if _, err := tk.Verify(bad); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("Verify(%q) err = %v, want ErrInvalidToken", bad, err)
		}
	}
}

func TestTokenExpiry(t *testing.T) {
	issuer := NewTokens("secret-a", time.Hour, fixedClock(t0))
	token, _ := issuer.Issue("admin")

	before := NewTokens("secret-a", time.Hour, fixedClock(t0.Add(59*time.Minute)))
	if _, err := before.Verify(token); err != nil {
		t.Fatalf("before expiry: %v", err)
	}
	after := NewTokens("secret-a", time.Hour, fixedClock(t0.Add(time.Hour)))
	if _, err := after.Verify(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("at expiry: err = %v, want ErrExpiredToken", err)
	}
}

func TestTokenSurvivesRestartWithSameSecret(t *testing.T) {
	token, _ := NewTokens("secret-a", time.Hour, fixedClock(t0)).Issue("admin")
	restarted := NewTokens("secret-a", time.Hour, fixedClock(t0.Add(10*time.Minute)))
	if _, err := restarted.Verify(token); err != nil {
		t.Fatalf("token should stay valid after restart: %v", err)
	}
	other := NewTokens("secret-b", time.Hour, fixedClock(t0))
	if _, err := other.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("different secret: err = %v, want ErrInvalidToken", err)
	}
}
