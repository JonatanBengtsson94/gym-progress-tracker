package config

import "os"

type Config struct {
	DB PostgresConfig
}

type PostgresConfig struct {
	Username string
	Password string
	Host     string
	Port     string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		DB: PostgresConfig{
			Username: os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PWD"),
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
		},
	}

	return cfg, nil
}
