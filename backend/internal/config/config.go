package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	DB      PostgresConfig
	Session SessionConfig
}

type PostgresConfig struct {
	DBName   string
	Username string
	Password string
	Host     string
	Port     string
}

type SessionConfig struct {
	SessionDuration time.Duration
}

func LoadConfig() (*Config, error) {
	sessionDurationStr := os.Getenv("SESSION_DURATION")
	if sessionDurationStr == "" {
		return nil, fmt.Errorf("SESSION_DURATION is not set")
	}

	sessionDuration, err := time.ParseDuration(sessionDurationStr)
	if err != nil {
		return nil, fmt.Errorf("SESSION_DURATION is invalid: %w", err)
	}

	db, err := LoadPostgresConfig()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		DB: db,
		Session: SessionConfig{
			SessionDuration: sessionDuration,
		},
	}

	return cfg, nil
}

// LoadPostgresConfig loads only the database settings, for tools that need a
// database connection but not the rest of the server config.
func LoadPostgresConfig() (PostgresConfig, error) {
	cfg := PostgresConfig{
		DBName:   os.Getenv("DB_NAME"),
		Username: os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PWD"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
	}

	if cfg.DBName == "" {
		return PostgresConfig{}, fmt.Errorf("DB_NAME is not set")
	}
	if cfg.Username == "" {
		return PostgresConfig{}, fmt.Errorf("DB_USER is not set")
	}
	if cfg.Password == "" {
		return PostgresConfig{}, fmt.Errorf("DB_PWD is not set")
	}
	if cfg.Host == "" {
		return PostgresConfig{}, fmt.Errorf("DB_HOST is not set")
	}
	if cfg.Port == "" {
		return PostgresConfig{}, fmt.Errorf("DB_Port is not set")
	}

	return cfg, nil
}
