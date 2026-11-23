package config

import (
	"net/url"
	"os"
	"testing"
)

func TestMigrationOverrideAndReadOnlyInstances(t *testing.T) {
	t.Setenv("PORT", "8081")
	t.Setenv("DB_MIGRATE", "")
	if err := os.Unsetenv("DB_MIGRATE"); err != nil {
		t.Fatal(err)
	}
	if LoadConfig().Migrate {
		t.Fatal("legacy read-only instance attempts migration")
	}
	t.Setenv("DB_MIGRATE", "true")
	if !LoadConfig().Migrate {
		t.Fatal("migration override on a custom port was ignored")
	}
	t.Setenv("PORT", "8080")
	t.Setenv("DB_MIGRATE", "false")
	if LoadConfig().Migrate {
		t.Fatal("disabled migration was ignored")
	}
}

func TestDSNEscapesCredentials(t *testing.T) {
	cfg := &Config{Host: "::1", Port: "5432", User: "user@shop", Password: "p@ss word'&?", DBName: "shopapi", SSLMode: "disable"}
	parsed, err := url.Parse(cfg.DSN())
	if err != nil {
		t.Fatal(err)
	}
	password, _ := parsed.User.Password()
	if password != cfg.Password || parsed.User.Username() != cfg.User || parsed.Host != "[::1]:5432" || parsed.Query().Get("sslmode") != "disable" {
		t.Fatal("connection parameters changed during URL encoding")
	}
}
