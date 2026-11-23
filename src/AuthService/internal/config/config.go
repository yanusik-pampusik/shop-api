package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	DatabaseDSN string
	GRPCAddress string
	JWTSecret   string
	JWTIssuer   string
	JWTAudience string
	TokenTTL    time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseDSN: os.Getenv("AUTH_DB_DSN"),
		GRPCAddress: envOr("AUTH_GRPC_ADDR", "127.0.0.1:50051"),
		JWTSecret:   os.Getenv("AUTH_JWT_SECRET"),
		JWTIssuer:   envOr("AUTH_JWT_ISSUER", "authservice"),
		JWTAudience: envOr("AUTH_JWT_AUDIENCE", "shopapi"),
	}
	if cfg.DatabaseDSN == "" {
		return Config{}, errors.New("AUTH_DB_DSN is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("AUTH_JWT_SECRET must contain at least 32 bytes")
	}
	ttl, err := time.ParseDuration(envOr("AUTH_TOKEN_TTL", "1h"))
	if err != nil || ttl < time.Second {
		return Config{}, errors.New("AUTH_TOKEN_TTL must be a duration of at least 1s")
	}
	cfg.TokenTTL = ttl
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
