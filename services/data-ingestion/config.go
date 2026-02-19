// Конфигурация data-ingestion (PostgreSQL, Pulsar).
package main

import "os"

// Config — настройки подключения к PostgreSQL и Pulsar.
type Config struct {
	PostgresConnStr string
	PulsarURL       string
}

// LoadConfig загружает конфиг из переменных окружения с дефолтами.
func LoadConfig() Config {
	cfg := Config{
		PostgresConnStr: os.Getenv("POSTGRES_CONN_STR"),
		PulsarURL:       os.Getenv("PULSAR_URL"),
	}
	// Значения по умолчанию, если env не заданы
	if cfg.PostgresConnStr == "" {
		cfg.PostgresConnStr = "postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
	}
	if cfg.PulsarURL == "" {
		cfg.PulsarURL = "pulsar://localhost:6650"
	}
	return cfg
}
