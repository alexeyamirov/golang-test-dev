// Package adapters содержит адаптеры для преобразования метрик TR181 в алерты.
package adapters

import (
	"golang-test-dev/pkg/tr181"
)

// AlertResult — результат оценки адаптера: нужен ли алерт
type AlertResult struct {
	Type  tr181.AlertType
	Value int
}

// Adapter оценивает данные устройства и возвращает алерты при необходимости.
// Adapter привязан к конкретной метрике/условию.
type Adapter interface {
	Evaluate(device *tr181.TR181Device) []AlertResult
}
