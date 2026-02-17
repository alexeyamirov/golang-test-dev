// Package pulsar предоставляет клиент для подключения к Apache Pulsar.
package pulsar

import (
	"os"
	"time"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
)

const (
	// TopicTR181Data — топик для данных устройств TR181 (метрики, телеметрия).
	TopicTR181Data = "persistent://public/default/tr181-device-data"
	// TopicAlerts — топик для алертов (устаревший, алерты пишутся в БД).
	TopicAlerts = "persistent://public/default/alerts"
	// TopicLogs — топик для логов (опциональный log-viewer подписывается).
	TopicLogs = "persistent://public/default/tr181-logs"
)

// NewClient создаёт Pulsar клиент.
func NewClient(url string) (pulsarclient.Client, error) {
	if url == "" {
		url = os.Getenv("PULSAR_URL")
	}
	if url == "" {
		url = "pulsar://localhost:6650"
	}

	return pulsarclient.NewClient(pulsarclient.ClientOptions{
		URL:               url,
		OperationTimeout:  90 * time.Second, // создание топика при первом producer может занимать 45–60 сек
		ConnectionTimeout: 30 * time.Second,
	})
}
