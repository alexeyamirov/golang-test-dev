package adapters

import (
	"golang-test-dev/pkg/tr181"
)

// cpuAlertThreshold — порог загрузки CPU (%), выше которого создаётся алерт.
const cpuAlertThreshold = 60

// CPUAdapter проверяет высокую загрузку CPU.
type CPUAdapter struct{}

// NewCPUAdapter создаёт адаптер для алерта high-cpu-usage.
func NewCPUAdapter() *CPUAdapter {
	return &CPUAdapter{}
}

// Evaluate оценивает данные устройства и возвращает алерты при CPU > 60%.
func (a *CPUAdapter) Evaluate(device *tr181.TR181Device) []AlertResult {
	if device.Data.CPUUsage <= cpuAlertThreshold {
		return nil // норма — алерт не нужен
	}
	return []AlertResult{
		{Type: tr181.AlertHighCPUUsage, Value: device.Data.CPUUsage},
	}
}
