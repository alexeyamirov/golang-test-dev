// Обработчик Pulsar сообщений для data-ingestion.
package main

import (
	"context"
	"fmt"
	"log"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/logcollector"
)

// MessageHandler парсит TR181 сообщения и сохраняет метрики.
type MessageHandler struct {
	storage   *MetricStorage
	consumer  pulsarclient.Consumer
	logColl   *logcollector.Collector
	processed int // счётчик для периодического лога
}

// NewMessageHandler создаёт обработчик.
func NewMessageHandler(storage *MetricStorage, consumer pulsarclient.Consumer, logColl *logcollector.Collector) *MessageHandler {
	return &MessageHandler{storage: storage, consumer: consumer, logColl: logColl}
}

// Handle парсит сообщение и сохраняет все метрики устройства в БД.
func (h *MessageHandler) Handle(ctx context.Context, msg pulsarclient.Message) {
	// Парсим JSON-тело сообщения
	device, err := ParseTR181Payload(msg.Payload())
	if err != nil {
		log.Printf("parse: %v", err)
		h.consumer.Nack(msg) // откатываем для повтора
		return
	}

	if device.SerialNumber == "" {
		h.consumer.Ack(msg)
		return
	}

	// Сохраняем все метрики (CPU, память, WiFi, Ethernet и т.д.)
	if err := h.storage.Save(ctx, device); err != nil {
		log.Printf("save: %v", err)
		h.consumer.Nack(msg)
		return
	}

	h.processed++
	// Каждые 50 устройств — лог в log-viewer
	if h.processed%50 == 0 && h.logColl != nil {
		h.logColl.Send("data-ingestion", "info",
			fmt.Sprintf("processed %d devices (last: %s)", h.processed, device.SerialNumber))
	}

	h.consumer.Ack(msg)
}
