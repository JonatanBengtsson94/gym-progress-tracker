package config

import (
	"fmt"
	"os"
)

type Config struct {
	DB PostgresConfig
}

type PostgresConfig struct {
	DBName   string
	Username string
	Password string
	Host     string
	Port     string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		DB: PostgresConfig{
			DBName:   os.Getenv("DB_NAME"),
			Username: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PWD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
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
