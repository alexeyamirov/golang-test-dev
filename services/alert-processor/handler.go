// Обработчик сообщений Pulsar для alert-processor.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/tr181"
	"golang-test-dev/pkg/logcollector"
	"golang-test-dev/services/alert-processor/adapters"
)

// AlertHandler разбирает TR181 сообщения, оценивает через адаптеры и сохраняет алерты.
type AlertHandler struct {
	storage  *AlertStorage
	consumer pulsarclient.Consumer
	logColl  *logcollector.Collector
	adapters []adapters.Adapter
}

// NewAlertHandler создаёт обработчик с storage и списком адаптеров.
func NewAlertHandler(storage *AlertStorage, consumer pulsarclient.Consumer, logColl *logcollector.Collector) *AlertHandler {
	return &AlertHandler{
		storage:  storage,
		consumer: consumer,
		logColl:  logColl,
		adapters: adapters.Registry(),
	}
}

// Handle парсит сообщение, прогоняет через адаптеры и сохраняет алерты.
func (h *AlertHandler) Handle(ctx context.Context, msg pulsarclient.Message) {
	var device tr181.TR181Device
	// Парсим JSON в структуру TR181Device
	if err := json.Unmarshal(msg.Payload(), &device); err != nil {
		log.Printf("parse: %v", err)
		h.consumer.Ack(msg) // подтверждаем, чтобы не получать повторно
		return
	}

	if device.SerialNumber == "" {
		h.consumer.Ack(msg)
		return // пропускаем сообщения без серийного номера
	}

	// Заполняем время, если не указано
	if device.Timestamp.IsZero() {
		device.Timestamp = time.Now()
	}

	// Прогоняем через все адаптеры (CPU, WiFi и т.д.)
	for _, a := range h.adapters {
		results := a.Evaluate(&device)
		for _, r := range results {
			// Сохраняем каждый алерт в PostgreSQL
			if err := h.storage.Save(ctx, device.SerialNumber, string(r.Type), r.Value, device.Timestamp); err != nil {
				log.Printf("save alert: %v", err)
				h.consumer.Nack(msg) // откатываем для повтора
				return
			}
			// Отправляем в log-viewer (если подключён)
			if h.logColl != nil {
				alertMsg := fmt.Sprintf("%s %s value=%d", device.SerialNumber, r.Type, r.Value)
				level := alertLevel(string(r.Type), r.Value) // warning или error по порогу
				h.logColl.Send("alert-processor", level, alertMsg)
			}
		}
	}

	h.consumer.Ack(msg)
}

// alertLevel определяет уровень алерта: error = критичное, warning = превышение нормы.
func alertLevel(alertType string, value int) string {
	switch alertType {
	case "high-cpu-usage":
		if value >= 80 {  // CPU 80%+ — критично
			return "error"
		}
		return "warning"  // 60-79% — предупреждение
	case "low-wifi":
		if value <= -110 {  // сигнал хуже -110 dBm — критично
			return "error"
		}
		return "warning"  // -100..-110 dBm — предупреждение
	default:
		return "warning"
	}
}
