// AlertStorage — слой сохранения алертов в PostgreSQL.
package main

import (
	"context"
	"time"

	"golang-test-dev/pkg/database"
)

// AlertStorage обёртка над PostgresDB для алертов.
type AlertStorage struct {
	db *database.PostgresDB
}

// NewAlertStorage создаёт storage для алертов.
func NewAlertStorage(db *database.PostgresDB) *AlertStorage {
	return &AlertStorage{db: db}
}

// Save сохраняет один алерт в БД.
func (s *AlertStorage) Save(ctx context.Context, serialNumber, alertType string, value int, ts time.Time) error {
	return s.db.SaveAlert(ctx, serialNumber, alertType, value, ts)
}
