package adapters

// Registry возвращает все зарегистрированные адаптеры для оценки TR181 данных.
// Новый адаптер — создать файл (например temperature_adapter.go) и добавить сюда.
func Registry() []Adapter {
	return []Adapter{
		NewCPUAdapter(),   // алерт при высокой загрузке CPU
		NewWiFiAdapter(),  // алерт при слабом сигнале WiFi
	}
}
