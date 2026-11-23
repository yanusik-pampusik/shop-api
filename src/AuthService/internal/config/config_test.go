package config

import (
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) {
	t.Helper()
	t.Setenv("AUTH_DB_DSN", "host=localhost dbname=test")
	t.Setenv("AUTH_JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("AUTH_GRPC_ADDR", "")
	t.Setenv("AUTH_JWT_ISSUER", "")
	t.Setenv("AUTH_JWT_AUDIENCE", "")
	t.Setenv("AUTH_TOKEN_TTL", "")
}

func TestLoad(t *testing.T) {
	setup(t)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GRPCAddress != "127.0.0.1:50051" || cfg.TokenTTL != time.Hour || cfg.JWTIssuer != "authservice" || cfg.JWTAudience != "shopapi" {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	t.Setenv("AUTH_GRPC_ADDR", "0.0.0.0:9000")
	t.Setenv("AUTH_TOKEN_TTL", "30m")
	t.Setenv("AUTH_JWT_ISSUER", "custom-auth")
	t.Setenv("AUTH_JWT_AUDIENCE", "custom-shop")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.GRPCAddress != "0.0.0.0:9000" || cfg.TokenTTL != 30*time.Minute || cfg.JWTIssuer != "custom-auth" || cfg.JWTAudience != "custom-shop" {
		t.Fatal("overrides ignored")
	}
}

func TestInvalidConfig(t *testing.T) {
	for _, tc := range []struct{ key, value string }{
		{"AUTH_DB_DSN", ""}, {"AUTH_JWT_SECRET", ""}, {"AUTH_JWT_SECRET", "short"},
		{"AUTH_TOKEN_TTL", "nope"}, {"AUTH_TOKEN_TTL", "-1h"}, {"AUTH_TOKEN_TTL", "0"}, {"AUTH_TOKEN_TTL", "1ms"},
	} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			setup(t)
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
