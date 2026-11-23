package config

import (
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config хранит настройки подключения к БД.
type Config struct {
	Host        string
	Port        string
	User        string
	Password    string
	DBName      string
	SSLMode     string
	HTTPAddress string
	AuthAddress string
	AuthTimeout time.Duration
	Migrate     bool
}

// LoadConfig читает настройки из переменных окружения.
// Если переменная не задана — использует значение по умолчанию.
func LoadConfig() *Config {
	return &Config{
		Host:        getEnv("DB_HOST", "localhost"),
		Port:        getEnv("DB_PORT", "5432"),
		User:        getEnv("DB_USER", "postgres"),
		Password:    getEnv("DB_PASSWORD", "postgres"),
		DBName:      getEnv("DB_NAME", "shopapi"),
		SSLMode:     getEnv("DB_SSLMODE", "disable"),
		HTTPAddress: ":" + getEnv("PORT", "8080"),
		AuthAddress: getEnv("AUTH_GRPC_ADDR", "127.0.0.1:50051"),
		AuthTimeout: 3 * time.Second,
		// Keep the previous project's read-only instances on other ports.
		Migrate: getEnv("DB_MIGRATE", strconv.FormatBool(getEnv("PORT", "8080") == "8080")) == "true",
	}
}

// DSN собирает строку подключения для GORM.
func (c *Config) DSN() string {
	u := url.URL{Scheme: "postgres", User: url.UserPassword(c.User, c.Password),
		Host: net.JoinHostPort(c.Host, c.Port), Path: "/" + c.DBName}
	query := url.Values{"sslmode": {c.SSLMode}}
	u.RawQuery = query.Encode()
	return u.String()
}

// ConnectDB открывает соединение с PostgreSQL через GORM.
func ConnectDB(cfg *Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}
	log.Println("Подключение к PostgreSQL установлено")
	return db, nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
