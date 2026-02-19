// Парсер TR181 JSON для data-ingestion.
package main

import (
	"encoding/json"
	"time"

	"golang-test-dev/pkg/tr181"
)

// ParseTR181Payload парсит JSON в TR181Device; заполняет Timestamp при необходимости.
func ParseTR181Payload(payload []byte) (*tr181.TR181Device, error) {
	var device tr181.TR181Device
	if err := json.Unmarshal(payload, &device); err != nil {
		return nil, err
	}
	// Если время не указано — подставляем текущее
	if device.Timestamp.IsZero() {
		device.Timestamp = time.Now()
	}
	return &device, nil
}
