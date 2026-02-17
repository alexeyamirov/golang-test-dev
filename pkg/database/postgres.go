// Package database — см. package doc в redis.go.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// PostgresDB — обёртка над sql.DB для метрик и алертов (TimescaleDB).
type PostgresDB struct {
	db *sql.DB
}

// NewPostgresDB подключается к PostgreSQL и возвращает клиент БД.
func NewPostgresDB(connStr string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresDB{db: db}, nil
}

// Close закрывает соединение с БД.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// DB возвращает низкоуровневый *sql.DB (для миграций и т.п.).
func (p *PostgresDB) DB() *sql.DB {
	return p.db
}

// InitSchema создает необходимые таблицы
func (p *PostgresDB) InitSchema(ctx context.Context) error {
	queries := []string{
		// Расширение для TimescaleDB (если доступно)
		`CREATE EXTENSION IF NOT EXISTS timescaledb;`,
		
		// Таблица для метрик
		`CREATE TABLE IF NOT EXISTS metrics (
			id BIGSERIAL PRIMARY KEY,
			serial_number VARCHAR(255) NOT NULL,
			metric_type VARCHAR(100) NOT NULL,
			value INTEGER NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,
		
		// Создание hypertable для TimescaleDB
		`SELECT create_hypertable('metrics', 'timestamp', if_not_exists => TRUE);`,
		
		// Индексы для быстрого поиска
		`CREATE INDEX IF NOT EXISTS idx_metrics_serial_time ON metrics(serial_number, timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_type_time ON metrics(metric_type, timestamp DESC);`,
		
		// Таблица для алертов
		`CREATE TABLE IF NOT EXISTS alerts (
			id BIGSERIAL PRIMARY KEY,
			serial_number VARCHAR(255) NOT NULL,
			alert_type VARCHAR(100) NOT NULL,
			value INTEGER NOT NULL,
			timestamp TIMESTAMPTZ NOT NULL,
			processed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`,
		
		`CREATE INDEX IF NOT EXISTS idx_alerts_serial_time ON alerts(serial_number, timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_type_time ON alerts(alert_type, timestamp DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_processed ON alerts(processed) WHERE processed = FALSE;`,
	}

	for _, query := range queries {
		if _, err := p.db.ExecContext(ctx, query); err != nil {
			// Игнорируем ошибку если TimescaleDB недоступен
			if query == `SELECT create_hypertable('metrics', 'timestamp', if_not_exists => TRUE);` {
				continue
			}
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// SaveMetric сохраняет метрику в БД
func (p *PostgresDB) SaveMetric(ctx context.Context, serialNumber, metricType string, value int, timestamp time.Time) error {
	query := `INSERT INTO metrics (serial_number, metric_type, value, timestamp) VALUES ($1, $2, $3, $4)`
	_, err := p.db.ExecContext(ctx, query, serialNumber, metricType, value, timestamp)
	return err
}

// GetMetrics получает метрики за период
func (p *PostgresDB) GetMetrics(ctx context.Context, serialNumber, metricType string, from, to time.Time) ([]MetricValue, error) {
	query := `SELECT value, EXTRACT(EPOCH FROM timestamp)::BIGINT as time 
			  FROM metrics 
			  WHERE serial_number = $1 AND metric_type = $2 AND timestamp >= $3 AND timestamp <= $4 
			  ORDER BY timestamp ASC`
	
	rows, err := p.db.QueryContext(ctx, query, serialNumber, metricType, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []MetricValue
	for rows.Next() {
		var m MetricValue
		if err := rows.Scan(&m.Value, &m.Time); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// SaveAlert сохраняет алерт в БД
func (p *PostgresDB) SaveAlert(ctx context.Context, serialNumber, alertType string, value int, timestamp time.Time) error {
	query := `INSERT INTO alerts (serial_number, alert_type, value, timestamp) VALUES ($1, $2, $3, $4)`
	_, err := p.db.ExecContext(ctx, query, serialNumber, alertType, value, timestamp)
	return err
}

// GetAlertStats получает статистику по алертам
func (p *PostgresDB) GetAlertStats(ctx context.Context, serialNumber, alertType string, from, to time.Time) (*AlertStats, error) {
	query := `SELECT AVG(value)::INTEGER as avg_value, COUNT(*) as count 
			  FROM alerts 
			  WHERE serial_number = $1 AND alert_type = $2 AND timestamp >= $3 AND timestamp <= $4`
	
	var stats AlertStats
	err := p.db.QueryRowContext(ctx, query, serialNumber, alertType, from, to).Scan(&stats.Value, &stats.Count)
	if err == sql.ErrNoRows {
		return &AlertStats{Value: 0, Count: 0}, nil
	}
	return &stats, err
}

// MetricValue — значение метрики с временной меткой (используется в pkg/database).
type MetricValue struct {
	Value int   `json:"value"`
	Time  int64 `json:"time"`
}

// AlertStats — агрегированная статистика алертов (среднее значение и количество).
type AlertStats struct {
	Value int `json:"value"`
	Count int `json:"count"`
}
