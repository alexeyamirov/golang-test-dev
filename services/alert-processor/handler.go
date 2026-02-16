// Обработчик сообщений Pulsar для alert-processor.
package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/tr181"
	"golang-test-dev/services/alert-processor/adapters"
)

// AlertHandler разбирает TR181 сообщения, оценивает через адаптеры и сохраняет алерты.
type AlertHandler struct {
	storage   *AlertStorage
	consumer  pulsarclient.Consumer
	adapters  []adapters.Adapter
}

// NewAlertHandler создаёт обработчик с storage и списком адаптеров.
func NewAlertHandler(storage *AlertStorage, consumer pulsarclient.Consumer) *AlertHandler {
	return &AlertHandler{
		storage:  storage,
		consumer: consumer,
		adapters: adapters.Registry(),
	}
}

// Handle парсит сообщение, прогоняет через адаптеры и сохраняет алерты.
func (h *AlertHandler) Handle(ctx context.Context, msg pulsarclient.Message) {
	var device tr181.TR181Device
	if err := json.Unmarshal(msg.Payload(), &device); err != nil {
		log.Printf("parse: %v", err)
		h.consumer.Ack(msg)
		return
	}

	if device.SerialNumber == "" {
		h.consumer.Ack(msg)
		return
	}

	if device.Timestamp.IsZero() {
		device.Timestamp = time.Now()
	}

	for _, a := range h.adapters {
		results := a.Evaluate(&device)
		for _, r := range results {
			if err := h.storage.Save(ctx, device.SerialNumber, string(r.Type), r.Value, device.Timestamp); err != nil {
				log.Printf("save alert: %v", err)
				h.consumer.Nack(msg)
				return
			}
		}
	}

	h.consumer.Ack(msg)
}
