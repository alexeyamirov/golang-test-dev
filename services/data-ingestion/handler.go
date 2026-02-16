// Обработчик Pulsar сообщений для data-ingestion.
package main

import (
	"context"
	"log"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
)

// MessageHandler парсит TR181 сообщения и сохраняет метрики.
type MessageHandler struct {
	storage  *MetricStorage
	consumer pulsarclient.Consumer
}

// NewMessageHandler создаёт обработчик.
func NewMessageHandler(storage *MetricStorage, consumer pulsarclient.Consumer) *MessageHandler {
	return &MessageHandler{storage: storage, consumer: consumer}
}

// Handle парсит сообщение и сохраняет все метрики устройства.
func (h *MessageHandler) Handle(ctx context.Context, msg pulsarclient.Message) {
	device, err := ParseTR181Payload(msg.Payload())
	if err != nil {
		log.Printf("parse: %v", err)
		h.consumer.Nack(msg)
		return
	}

	if device.SerialNumber == "" {
		h.consumer.Ack(msg)
		return
	}

	if err := h.storage.Save(ctx, device); err != nil {
		log.Printf("save: %v", err)
		h.consumer.Nack(msg)
		return
	}

	h.consumer.Ack(msg)
}
