package tr181

import "time"

// TR181Device представляет устройство с TR181 данными
type TR181Device struct {
	SerialNumber string    `json:"serial_number"`
	Timestamp    time.Time `json:"timestamp"`
	Data         DeviceData `json:"data"`
}

// DeviceData содержит основные параметры TR181 модели
type DeviceData struct {
	// Device.DeviceInfo.ProcessStatus
	CPUUsage    int `json:"Device.DeviceInfo.ProcessStatus.CPUUsage"`    // 0-100%
	MemoryUsage int `json:"Device.DeviceInfo.ProcessStatus.MemoryUsage"` // 0-100%
	
	// Device.DeviceInfo.Temperature
	CPUTemperature    int `json:"Device.DeviceInfo.Temperature.CPU"`     // градусы Цельсия
	BoardTemperature  int `json:"Device.DeviceInfo.Temperature.Board"`   // градусы Цельсия
	RadioTemperature  int `json:"Device.DeviceInfo.Temperature.Radio"`   // градусы Цельсия
	
	// Device.WiFi.AccessPoint.{i}.AssociatedDevice.{i}
	WiFi2GHzSignalStrength int `json:"Device.WiFi.AccessPoint.0.AssociatedDevice.0.SignalStrength"` // dBm
	WiFi5GHzSignalStrength  int `json:"Device.WiFi.AccessPoint.1.AssociatedDevice.0.SignalStrength"` // dBm
	WiFi6GHzSignalStrength  int `json:"Device.WiFi.AccessPoint.2.AssociatedDevice.0.SignalStrength"` // dBm
	
	// Device.Ethernet.Interface.{i}.Stats
	EthernetBytesSent   int64 `json:"Device.Ethernet.Interface.0.Stats.BytesSent"`
	EthernetBytesReceived int64 `json:"Device.Ethernet.Interface.0.Stats.BytesReceived"`
	
	// Device.DeviceInfo.UpTime
	Uptime int64 `json:"Device.DeviceInfo.UpTime"` // секунды
	
	// Customer Extensions (примеры)
	CustomField1 int `json:"Custom.Extension.Field1,omitempty"`
	CustomField2 int `json:"Custom.Extension.Field2,omitempty"`
}

// MetricType представляет тип метрики для маппинга
type MetricType string

const (
	MetricCPUUsage            MetricType = "cpu-usage"
	MetricMemoryUsage         MetricType = "memory-usage"
	MetricCPUTemperature      MetricType = "cpu-temperature"
	MetricBoardTemperature    MetricType = "board-temperature"
	MetricRadioTemperature    MetricType = "radio-temperature"
	MetricWiFi2GHzSignal      MetricType = "wifi-2ghz-signal"
	MetricWiFi5GHzSignal      MetricType = "wifi-5ghz-signal"
	MetricWiFi6GHzSignal      MetricType = "wifi-6ghz-signal"
	MetricEthernetBytesSent   MetricType = "ethernet-bytes-sent"
	MetricEthernetBytesRecv   MetricType = "ethernet-bytes-received"
	MetricUptime              MetricType = "uptime"
)

// AlertType представляет тип алерта
type AlertType string

const (
	AlertHighCPUUsage AlertType = "high-cpu-usage"
	AlertLowWiFi      AlertType = "low-wifi"
)

// MetricValue представляет значение метрики с временной меткой
type MetricValue struct {
	Value int   `json:"value"`
	Time  int64 `json:"time"` // Unix timestamp
}

// AlertData представляет данные алерта
type AlertData struct {
	Value int `json:"value"` // среднее значение за период
	Count int `json:"count"` // количество алертов за период
}

// GetMetricValue извлекает значение метрики из DeviceData
func (d *DeviceData) GetMetricValue(metricType MetricType) (int, bool) {
	switch metricType {
	case MetricCPUUsage:
		return d.CPUUsage, true
	case MetricMemoryUsage:
		return d.MemoryUsage, true
	case MetricCPUTemperature:
		return d.CPUTemperature, true
	case MetricBoardTemperature:
		return d.BoardTemperature, true
	case MetricRadioTemperature:
		return d.RadioTemperature, true
	case MetricWiFi2GHzSignal:
		return d.WiFi2GHzSignalStrength, true
	case MetricWiFi5GHzSignal:
		return d.WiFi5GHzSignalStrength, true
	case MetricWiFi6GHzSignal:
		return d.WiFi6GHzSignalStrength, true
	case MetricEthernetBytesSent:
		return int(d.EthernetBytesSent), true
	case MetricEthernetBytesRecv:
		return int(d.EthernetBytesReceived), true
	case MetricUptime:
		return int(d.Uptime), true
	default:
		return 0, false
	}
}

// CheckAlerts проверяет условия для алертов
func (d *DeviceData) CheckAlerts() []AlertType {
	var alerts []AlertType
	
	// High CPU usage alert
	if d.CPUUsage > 60 {
		alerts = append(alerts, AlertHighCPUUsage)
	}
	
	// Low WiFi signal alert (любой из WiFi сигналов)
	if d.WiFi2GHzSignalStrength < -100 || 
	   d.WiFi5GHzSignalStrength < -100 || 
	   d.WiFi6GHzSignalStrength < -100 {
		alerts = append(alerts, AlertLowWiFi)
	}
	
	return alerts
}
