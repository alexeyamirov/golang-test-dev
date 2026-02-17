// Package logcollector — опциональная публикация логов в Pulsar для log-viewer.
package logcollector

import (
	"context"
	"encoding/json"
	"os"
	"time"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/pulsar"
)

// LogEntry — структура сообщения в топике логов.
type LogEntry struct {
	Service   string `json:"service"`
	Level     string `json:"level"` // info, warn, error
	Message   string `json:"msg"`
	Timestamp int64  `json:"ts"`
}

// Collector публикует логи в Pulsar. Необязателен: при ошибке подключения Send — no-op.
type Collector struct {
	producer pulsarclient.Producer
	client   pulsarclient.Client // для Close, если создан здесь
	ownClient bool
}

// New создаёт collector с собственным Pulsar клиентом (для api-gateway). При ошибке — nil.
func New(serviceName string) *Collector {
	url := os.Getenv("PULSAR_URL")
	if url == "" {
		url = "pulsar://localhost:6650"
	}
	client, err := pulsar.NewClient(url)
	if err != nil {
		return nil
	}
	return NewFromClient(client, serviceName, true)
}

// NewFromClient создаёт collector из существующего клиента (для simulator, data-ingestion, alert-processor).
// serviceName — уникальное имя для producer (избегаем конфликтов). ownClient=false — клиент не закрывается при Close().
func NewFromClient(client pulsarclient.Client, serviceName string, ownClient bool) *Collector {
	name := "log-" + serviceName
	if name == "log-" {
		name = "log-collector"
	}
	prod, err := client.CreateProducer(pulsarclient.ProducerOptions{
		Topic: pulsar.TopicLogs,
		Name:  name,
	})
	if err != nil {
		if ownClient {
			client.Close()
		}
		return nil
	}
	return &Collector{producer: prod, client: client, ownClient: ownClient}
}


// Send публикует лог в топик. No-op если producer == nil.
func (c *Collector) Send(service, level, msg string) {
	if c == nil || c.producer == nil {
		return
	}
	entry := LogEntry{
		Service:   service,
		Level:     level,
		Message:   msg,
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_, _ = c.producer.Send(context.Background(), &pulsarclient.ProducerMessage{Payload: data})
}

// Close освобождает producer и при ownClient — клиент.
func (c *Collector) Close() error {
	if c == nil || c.producer == nil {
		return nil
	}
	c.producer.Close()
	if c.ownClient && c.client != nil {
		c.client.Close()
	}
	return nil
}
