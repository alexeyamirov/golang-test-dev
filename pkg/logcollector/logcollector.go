// Package logcollector — опциональная публикация логов в Pulsar для log-viewer.
package logcollector

import (
	"context"
	"encoding/json"
	"os"
	"time"

	"golang-test-dev/pkg/pulsar"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
)

// LogEntry — структура сообщения в топике логов.
type LogEntry struct {
	Service   string `json:"service"` // имя сервиса (simulator, data-ingestion и т.д.)
	Level     string `json:"level"`   // уровень: info, warn, error
	Message   string `json:"msg"`     // текст лога
	Timestamp int64  `json:"ts"`      // Unix-время
}

// Collector публикует логи в Pulsar. Необязателен: при ошибке подключения Send — no-op.
type Collector struct {
	producer  pulsarclient.Producer // producer для топика tr181-logs
	client    pulsarclient.Client   // клиент (закрываем при ownClient=true)
	ownClient bool                  // true — мы создали client, закрываем при Close
}

// New создает collector с собственным Pulsar клиентом (для api-gateway). При ошибке — nil.
func New(serviceName string) *Collector {
	url := os.Getenv("PULSAR_URL") // адрес брокера
	if url == "" {
		url = "pulsar://localhost:6650"
	}
	client, err := pulsar.NewClient(url)
	if err != nil {
		return nil // не удалось подключиться — collector не используется
	}
	return NewFromClient(client, serviceName, true) // ownClient=true — мы владеем client
}

// NewFromClient создает collector из существующего клиента (для simulator, data-ingestion, alert-processor).
// serviceName — уникальное имя producer. ownClient=false — клиент не закрывается при Close().
func NewFromClient(client pulsarclient.Client, serviceName string, ownClient bool) *Collector {
	name := "log-" + serviceName // уникальное имя producer
	if name == "log-" {
		name = "log-collector" // fallback при пустом имени
	}
	prod, err := client.CreateProducer(pulsarclient.ProducerOptions{
		Topic: pulsar.TopicLogs, // топик tr181-logs
		Name:  name,
	})
	if err != nil {
		if ownClient {
			client.Close() // освобождаем клиент при ошибке
		}
		return nil
	}
	return &Collector{producer: prod, client: client, ownClient: ownClient}
}

// Send публикует лог в топик. No-op если producer == nil.
func (c *Collector) Send(service, level, msg string) {
	if c == nil || c.producer == nil {
		return // collector не инициализирован
	}
	entry := LogEntry{
		Service:   service,
		Level:     level,
		Message:   msg,
		Timestamp: time.Now().Unix(), // текущее время в Unix
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return // игнорируем ошибку сериализации
	}
	_, _ = c.producer.Send(context.Background(), &pulsarclient.ProducerMessage{Payload: data})
}

// Close освобождает producer и при ownClient — клиент Pulsar.
func (c *Collector) Close() error {
	if c == nil || c.producer == nil {
		return nil
	}
	c.producer.Close() // закрываем producer
	if c.ownClient && c.client != nil {
		c.client.Close() // закрываем клиент, если мы его создавали
	}
	return nil
}
