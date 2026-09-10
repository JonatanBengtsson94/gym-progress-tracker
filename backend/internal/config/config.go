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

	cfg := &Config{
		DB: PostgresConfig{
			DBName:   os.Getenv("DB_NAME"),
			Username: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PWD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
		},
		Session: SessionConfig{
			SessionDuration: sessionDuration,
		},
	}

	if cfg.DB.DBName == "" {
		return nil, fmt.Errorf("DB_NAME is not set")
	}
	if cfg.DB.Username == "" {
		return nil, fmt.Errorf("DB_USER is not set")
	}
	if cfg.DB.Password == "" {
		return nil, fmt.Errorf("DB_PWD is not set")
	}
	if cfg.DB.Host == "" {
		return nil, fmt.Errorf("DB_HOST is not set")
	}
	if cfg.DB.Port == "" {
		return nil, fmt.Errorf("DB_Port is not set")
	}

	return cfg, nil
}
