// Пакет main - точка входа для сервиса приема данных (Data Ingestion Service)
// Потребляет TR181 данные из Apache Pulsar, сохраняет в PostgreSQL, публикует алерты в Pulsar
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
	"golang-test-dev/pkg/pulsar"                              // Наши константы тем Pulsar
	"golang-test-dev/pkg/tr181"                               // Модель данных TR181
)

func main() {
	// Строка подключения к PostgreSQL из переменной окружения
	postgresConnStr := os.Getenv("POSTGRES_CONN_STR")
	// Значение по умолчанию
	if postgresConnStr == "" {
		postgresConnStr = "postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
	}

	// Подключаемся к PostgreSQL
	postgresDB, err := database.NewPostgresDB(postgresConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	// Закрываем при выходе
	defer postgresDB.Close()

	// Контекст для инициализации
	ctx := context.Background()
	// Создаём таблицы в БД
	if err := postgresDB.InitSchema(ctx); err != nil {
		log.Printf("Warning: Schema initialization failed: %v", err)
	}

	// URL Apache Pulsar из переменной окружения
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

	// Создаём consumer - подписываемся на тему с данными устройств
	consumer, err := client.Subscribe(pulsarclient.ConsumerOptions{
		Topic:            pulsar.TopicTR181Data,    // persistent://public/default/tr181-device-data
		SubscriptionName: "data-ingestion-sub",     // Имя подписки (для группы потребителей)
		Type:             pulsarclient.Shared,      // Shared - сообщения распределяются между потребителями
	})
	if err != nil {
		log.Fatalf("Failed to create Pulsar consumer: %v", err)
	}
	defer consumer.Close()

	// Создаём producer - для отправки алертов в отдельную тему
	producer, err := client.CreateProducer(pulsarclient.ProducerOptions{
		Topic: pulsar.TopicAlerts,                   // persistent://public/default/alerts
		Name:  "data-ingestion-alert-producer",
	})
	if err != nil {
		log.Fatalf("Failed to create alert producer: %v", err)
	}
	defer producer.Close()

	log.Println("Data Ingestion Service started (consuming from Pulsar)")

	// Запускаем цикл приёма сообщений в отдельной горутине
	go func() {
		// Бесконечный цикл
		for {
			// Ожидаем следующее сообщение (блокирующий вызов)
			msg, err := consumer.Receive(context.Background())
			if err != nil {
				// При ошибке логируем и ждём секунду
				log.Printf("Consumer error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			// Обрабатываем полученные данные устройства
			processDeviceData(ctx, postgresDB, producer, consumer, msg)
		}
	}()

	// Канал для сигналов завершения
	quit := make(chan os.Signal, 1)
	// Регистрируем SIGINT и SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Ждём сигнал
	<-quit

	log.Println("Shutting down Data Ingestion Service...")
	log.Println("Server exited")
}

// processDeviceData - обрабатывает одно сообщение с данными TR181 устройства
func processDeviceData(ctx context.Context, postgresDB *database.PostgresDB, alertProducer pulsarclient.Producer, consumer pulsarclient.Consumer, msg pulsarclient.Message) {
	// Структура для десериализации JSON
	var device tr181.TR181Device
	// Парсим JSON из тела сообщения
	if err := json.Unmarshal(msg.Payload(), &device); err != nil {
		log.Printf("Failed to unmarshal TR181 data: %v", err)
		// Nack - сообщение вернётся для повторной обработки
		consumer.Nack(msg)
		return
	}

	// Проверка: серийный номер обязателен
	if device.SerialNumber == "" {
		log.Printf("Invalid message: missing serial_number")
		// Ack - подтверждаем (плохое сообщение не обрабатываем повторно)
		consumer.Ack(msg)
		return
	}

	// Если время не указано - используем текущее
	if device.Timestamp.IsZero() {
		device.Timestamp = time.Now()
	}

	// Список всех типов метрик для сохранения в БД
	metricTypes := []tr181.MetricType{
		tr181.MetricCPUUsage,
		tr181.MetricMemoryUsage,
		tr181.MetricCPUTemperature,
		tr181.MetricBoardTemperature,
		tr181.MetricRadioTemperature,
		tr181.MetricWiFi2GHzSignal,
		tr181.MetricWiFi5GHzSignal,
		tr181.MetricWiFi6GHzSignal,
		tr181.MetricEthernetBytesSent,
		tr181.MetricEthernetBytesRecv,
		tr181.MetricUptime,
	}

	// Сохраняем каждую метрику в PostgreSQL
	for _, mt := range metricTypes {
		// Получаем значение метрики из данных устройства
		if value, ok := device.Data.GetMetricValue(mt); ok {
			// Сохраняем в таблицу metrics
			if err := postgresDB.SaveMetric(ctx, device.SerialNumber, string(mt), value, device.Timestamp); err != nil {
				log.Printf("Failed to save metric %s: %v", mt, err)
			}
		}
	}

	// Проверяем условия алертов (высокий CPU, слабый WiFi)
	alerts := device.Data.CheckAlerts()
	// Для каждого сработавшего алерта - публикуем в Pulsar
	for _, alertType := range alerts {
		// Определяем значение для алерта
		var alertValue int
		switch alertType {
		case tr181.AlertHighCPUUsage:
			alertValue = device.Data.CPUUsage
		case tr181.AlertLowWiFi:
			// Берём минимальный сигнал из всех WiFi диапазонов
			alertValue = device.Data.WiFi2GHzSignalStrength
			if device.Data.WiFi5GHzSignalStrength < alertValue {
				alertValue = device.Data.WiFi5GHzSignalStrength
			}
			if device.Data.WiFi6GHzSignalStrength < alertValue {
				alertValue = device.Data.WiFi6GHzSignalStrength
			}
		}

		// Формируем JSON для алерта
		alertData := map[string]interface{}{
			"serial_number": device.SerialNumber,
			"alert_type":    string(alertType),
			"value":         alertValue,
			"timestamp":     device.Timestamp.Format(time.RFC3339),
		}
		alertJSON, _ := json.Marshal(alertData)

		// Отправляем в тему alerts для обработки Alert Processor
		_, err := alertProducer.Send(context.Background(), &pulsarclient.ProducerMessage{
			Payload: alertJSON,
			Key:     device.SerialNumber, // Ключ для партиционирования
		})
		if err != nil {
			log.Printf("Failed to publish alert: %v", err)
		}
	}

	// Подтверждаем успешную обработку сообщения
	consumer.Ack(msg)
	// Логируем если были алерты
	if len(alerts) > 0 {
		log.Printf("Processed device %s (%d alerts)", device.SerialNumber, len(alerts))
	}
}
