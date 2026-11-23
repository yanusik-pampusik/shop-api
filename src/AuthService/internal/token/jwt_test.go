package token

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestIssueAndValidate(t *testing.T) {
	manager, err := New(strings.Repeat("s", 32), "authservice", "shopapi", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1800000000, 0)
	manager.now = func() time.Time { return now }
	userID := uuid.NewString()
	raw, err := manager.Issue(userID, "user@example.com")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Validate(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != userID || claims.Email != "user@example.com" || claims.ID == "" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if !claims.ExpiresAt.Time.Equal(now.Add(time.Hour)) {
		t.Fatal("wrong expiration")
	}
	manager.now = func() time.Time { return now.Add(time.Hour) }
	if _, err := manager.Validate(raw); err != ErrInvalidToken {
		t.Fatalf("expired token accepted: %v", err)
	}
}

func TestRejectInvalidTokens(t *testing.T) {
	secret := strings.Repeat("s", 32)
	manager, _ := New(secret, "authservice", "shopapi", time.Hour)
	now := time.Unix(1800000000, 0)
	manager.now = func() time.Time { return now }
	cases := []struct {
		name   string
		mutate func(jwt.MapClaims)
		method jwt.SigningMethod
		key    any
	}{
		{name: "wrong key", key: []byte(strings.Repeat("x", 32))},
		{name: "wrong algorithm", method: jwt.SigningMethodHS512},
		{name: "unsigned", method: jwt.SigningMethodNone, key: jwt.UnsafeAllowNoneSignatureType},
		{name: "expired", mutate: func(c jwt.MapClaims) { c["exp"] = now.Unix() - 1 }},
		{name: "missing expiration", mutate: func(c jwt.MapClaims) { delete(c, "exp") }},
		{name: "wrong issuer", mutate: func(c jwt.MapClaims) { c["iss"] = "other" }},
		{name: "wrong audience", mutate: func(c jwt.MapClaims) { c["aud"] = "other" }},
		{name: "not yet valid", mutate: func(c jwt.MapClaims) { c["nbf"] = now.Unix() + 60 }},
		{name: "future issued at", mutate: func(c jwt.MapClaims) { c["iat"] = now.Unix() + 60 }},
		{name: "missing subject", mutate: func(c jwt.MapClaims) { delete(c, "sub") }},
		{name: "invalid user ID", mutate: func(c jwt.MapClaims) { c["sub"] = "not-a-uuid" }},
		{name: "nil user ID", mutate: func(c jwt.MapClaims) { c["sub"] = uuid.Nil.String() }},
		{name: "missing email", mutate: func(c jwt.MapClaims) { delete(c, "email") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": "authservice", "aud": "shopapi", "sub": uuid.NewString(),
				"email": "user@example.com", "exp": now.Add(time.Hour).Unix(), "iat": now.Unix()}
			if tc.mutate != nil {
				tc.mutate(claims)
			}
			method := tc.method
			if method == nil {
				method = jwt.SigningMethodHS256
			}
			key := tc.key
			if key == nil {
				key = []byte(secret)
			}
			raw, err := jwt.NewWithClaims(method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := manager.Validate(raw); err != ErrInvalidToken {
				t.Fatalf("invalid token accepted: %v", err)
			}
		})
	}
	for _, raw := range []string{"", "broken", "a.b.c"} {
		if _, err := manager.Validate(raw); err != ErrInvalidToken {
			t.Fatalf("malformed token accepted: %q", raw)
		}
	}
	raw, _ := manager.Issue(uuid.NewString(), "user@example.com")
	parts := strings.Split(raw, ".")
	parts[1] = parts[1][:len(parts[1])-1] + "A"
	if _, err := manager.Validate(strings.Join(parts, ".")); err != ErrInvalidToken {
		t.Fatal("tampered token accepted")
	}
}

func TestInvalidConfigurationAndIdentity(t *testing.T) {
	for _, cfg := range []struct {
		secret, issuer, audience string
		ttl                      time.Duration
	}{
		{"short", "auth", "shop", time.Hour},
		{strings.Repeat("s", 32), "", "shop", time.Hour},
		{strings.Repeat("s", 32), "auth", "", time.Hour},
		{strings.Repeat("s", 32), "auth", "shop", 0},
	} {
		if _, err := New(cfg.secret, cfg.issuer, cfg.audience, cfg.ttl); err == nil {
			t.Fatal("invalid config accepted")
		}
	}
	manager, _ := New(strings.Repeat("s", 32), "auth", "shop", time.Hour)
	if _, err := manager.Issue("", "user@example.com"); err == nil {
		t.Fatal("empty user accepted")
	}
	if _, err := manager.Issue(uuid.NewString(), ""); err == nil {
		t.Fatal("empty email accepted")
	}
}
