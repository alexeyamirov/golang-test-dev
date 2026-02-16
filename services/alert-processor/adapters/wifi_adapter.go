package adapters

import (
	"golang-test-dev/pkg/tr181"
)

// wifiAlertThreshold — порог сигнала WiFi (dBm), ниже которого создаётся алерт.
const wifiAlertThreshold = -100

// WiFiAdapter проверяет слабый сигнал WiFi (любой диапазон 2.4/5/6 GHz).
type WiFiAdapter struct{}

// NewWiFiAdapter создаёт адаптер для алерта low-wifi.
func NewWiFiAdapter() *WiFiAdapter {
	return &WiFiAdapter{}
}

// Evaluate оценивает данные устройства и возвращает алерт при сигнале < -100 dBm.
func (a *WiFiAdapter) Evaluate(device *tr181.TR181Device) []AlertResult {
	d := &device.Data
	if d.WiFi2GHzSignalStrength >= wifiAlertThreshold &&
		d.WiFi5GHzSignalStrength >= wifiAlertThreshold &&
		d.WiFi6GHzSignalStrength >= wifiAlertThreshold {
		return nil
	}

	value := d.WiFi2GHzSignalStrength
	if d.WiFi5GHzSignalStrength < value {
		value = d.WiFi5GHzSignalStrength
	}
	if d.WiFi6GHzSignalStrength < value {
		value = d.WiFi6GHzSignalStrength
	}

	return []AlertResult{
		{Type: tr181.AlertLowWiFi, Value: value},
	}
}
