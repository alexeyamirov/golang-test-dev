// Пакет main - точка входа для сервиса обработки алертов (Alert Processor)
// Потребляет алерты из Apache Pulsar и сохраняет в PostgreSQL
package main

import (
	"context"      // Контекст для отмены операций
	"encoding/json" // Парсинг JSON из сообщений Pulsar
	"log"          // Логирование
	"os"           // Переменные окружения
	"os/signal"    // Обработка сигналов завершения
	"syscall"      // SIGINT, SIGTERM
	"time"         // Работа со временем

	pulsarclient "github.com/apache/pulsar-client-go/pulsar" // Pulsar клиент
	"golang-test-dev/pkg/database"                            // PostgreSQL
	"golang-test-dev/pkg/pulsar"                              // Константы тем Pulsar
)

func main() {
	// Строка подключения к PostgreSQL из переменной окружения
	postgresConnStr := os.Getenv("POSTGRES_CONN_STR")
	if postgresConnStr == "" {
		postgresConnStr = "postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
	}

	// Подключаемся к PostgreSQL
	postgresDB, err := database.NewPostgresDB(postgresConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer postgresDB.Close()

	// Контекст для инициализации схемы БД
	ctx := context.Background()
	// Создаём таблицы если не существуют
	if err := postgresDB.InitSchema(ctx); err != nil {
		log.Printf("Warning: Schema initialization failed: %v", err)
	}

	// URL Apache Pulsar
	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		pulsarURL = "pulsar://localhost:6650"
	}

	// Подключаемся к Pulsar
	client, err := pulsar.NewClient(pulsarURL)
	if err != nil {
		log.Fatalf("Failed to connect to Pulsar: %v", err)
	}
	defer client.Close()

	// Создаём consumer - подписываемся на тему с алертами
	consumer, err := client.Subscribe(pulsarclient.ConsumerOptions{
		Topic:            pulsar.TopicAlerts,       // persistent://public/default/alerts
		SubscriptionName: "alert-processor-sub",    // Имя подписки
		Type:             pulsarclient.Shared,      // Shared - распределение между несколькими инстансами
	})
	if err != nil {
		log.Fatalf("Failed to create Pulsar consumer: %v", err)
	}
	defer consumer.Close()

	log.Println("Alert Processor started (consuming from Pulsar)")

	// Запускаем цикл обработки алертов в горутине
	go func() {
		for {
			// Ждём следующее сообщение
			msg, err := consumer.Receive(context.Background())
			if err != nil {
				log.Printf("Consumer error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			// Обрабатываем алерт
			processAlert(ctx, postgresDB, consumer, msg)
		}
	}()

	// Горутина для имитации CPU нагрузки (для автоскейлинга)
	go cpuLoadLoop()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down alert processor...")
}

// processAlert - обрабатывает одно сообщение с алертом
func processAlert(ctx context.Context, postgresDB *database.PostgresDB, consumer pulsarclient.Consumer, msg pulsarclient.Message) {
	// Map для парсинга JSON (serial_number, alert_type, value, timestamp)
	var alertData map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &alertData); err != nil {
		log.Printf("Failed to unmarshal alert: %v", err)
		// Подтверждаем - невалидное сообщение не переобрабатываем
		consumer.Ack(msg)
		return
	}

	// Извлекаем поля с проверкой типа (type assertion)
	serialNumber, _ := alertData["serial_number"].(string)
	alertType, _ := alertData["alert_type"].(string)

	// Value может прийти как float64 (из JSON) или int
	var value float64
	switch v := alertData["value"].(type) {
	case float64:
		value = v
	case int:
		value = float64(v)
	}

	// Парсим timestamp из строки
	var timestamp time.Time
	if timestampStr, ok := alertData["timestamp"].(string); ok {
		var err error
		timestamp, err = time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			// Пробуем альтернативный формат
			timestamp, err = time.Parse("2006-01-02T15:04:05Z07:00", timestampStr)
			if err != nil {
				timestamp = time.Now()
			}
		}
	} else {
		timestamp = time.Now()
	}

	// Сохраняем алерт в таблицу alerts в PostgreSQL
	if err := postgresDB.SaveAlert(ctx, serialNumber, alertType, int(value), timestamp); err != nil {
		log.Printf("Failed to save alert: %v", err)
		// Nack - сообщение вернётся для повторной попытки
		consumer.Nack(msg)
		return
	}

	// Имитация CPU нагрузки при обработке
	sum := 0
	for i := 0; i < 5000; i++ {
		sum += i * i
	}
	_ = sum

	// Подтверждаем успешную обработку
	consumer.Ack(msg)
	log.Printf("Processed alert: %s for device %s", alertType, serialNumber)
}

// cpuLoadLoop - создаёт постоянную небольшую CPU нагрузку (для тестирования автоскейлинга)
func cpuLoadLoop() {
	// Таймер срабатывает каждые 100 мс
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	// Бесконечный цикл
	for range ticker.C {
		sum := 0
		for i := 0; i < 10000; i++ {
			sum += i * i
		}
		_ = sum
	}
}
