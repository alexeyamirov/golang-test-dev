package adapters

// Registry возвращает все зарегистрированные адаптеры для оценки TR181 данных.
// Новый адаптер — новый файл и регистрация здесь.
func Registry() []Adapter {
	return []Adapter{
		NewCPUAdapter(),
		NewWiFiAdapter(),
	}
}
