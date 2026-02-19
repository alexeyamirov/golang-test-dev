// MetricStorage — слой сохранения метрик в PostgreSQL.
package main

import (
	"context"
	"log"

	"golang-test-dev/pkg/database"
	"golang-test-dev/pkg/tr181"
)

// metricTypes — список метрик, сохраняемых для каждого устройства.
var metricTypes = []tr181.MetricType{
	tr181.MetricCPUUsage,
	tr181.MetricMemoryUsage,
	tr181.MetricCPUTemperature,
	tr181.MetricBoardTemperature,
	tr181.MetricRadioTemperature,
	tr181.MetricWiFi2GHzSignal,
	tr181.MetricWiFi5GHzSignal,
	tr181.MetricWiFi6GHzSignal,
	tr181.MetricEthernetBytesSent,
	tr181.MetricEthernetBytesRecv,
	tr181.MetricUptime,
}

// MetricStorage обёртка над PostgresDB для массового сохранения метрик.
type MetricStorage struct {
	db *database.PostgresDB
}

// NewMetricStorage создаёт storage для метрик.
func NewMetricStorage(db *database.PostgresDB) *MetricStorage {
	return &MetricStorage{db: db}
}

// Save сохраняет все метрики устройства в БД (по одной записи на каждый тип).
func (s *MetricStorage) Save(ctx context.Context, device *tr181.TR181Device) error {
	for _, mt := range metricTypes {
		value, ok := device.Data.GetMetricValue(mt)
		if !ok {
			continue // метрика отсутствует в данных
		}
		// Вставляем в таблицу metrics
		if err := s.db.SaveMetric(ctx, device.SerialNumber, string(mt), value, device.Timestamp); err != nil {
			log.Printf("save metric %s: %v", mt, err)
		}
	}
	return nil
}
