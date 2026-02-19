// log-viewer — опциональный сервис: подписывается на топик логов и выводит все записи в один терминал.
// Можно запустить в любой момент после старта остальных сервисов.
package main

import (
	"context"      // Контекст для Receive
	"encoding/json" // Парсинг JSON из сообщений Pulsar
	"fmt"          // Вывод в stdout
	"log"          // Логирование
	"os"           // Переменные окружения
	"os/signal"    // Обработка Ctrl+C
	"strconv"      // Atoi для LOG_VIEWER_DELAY_MS
	"syscall"      // SIGINT, SIGTERM
	"time"         // Задержки, time.Unix

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/logcollector"
	"golang-test-dev/pkg/pulsar"
)

func main() {
	// Адрес Pulsar из env или localhost
	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		pulsarURL = "pulsar://localhost:6650"
	}

	// Подключаемся к Pulsar
	client, err := pulsar.NewClient(pulsarURL)
	if err != nil {
		log.Fatalf("pulsar: %v", err)
	}
	defer client.Close()

	// Подписываемся на топик логов (Latest — только новые, не backlog)
	consumer, err := client.Subscribe(pulsarclient.ConsumerOptions{
		Topic:            pulsar.TopicLogs,
		SubscriptionName: "log-viewer-sub",
		Type:             pulsarclient.Exclusive,
		SubscriptionInitialPosition: pulsarclient.SubscriptionPositionLatest,
	})
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	// Задержка между строками (мс) — чтобы успевать читать
	delayMs, _ := strconv.Atoi(os.Getenv("LOG_VIEWER_DELAY_MS"))
	if delayMs <= 0 {
		delayMs = 1000 // по умолчанию 1000 мс (~1 строка/сек). LOG_VIEWER_DELAY_MS=500 — быстрее
	}
	delay := time.Duration(delayMs) * time.Millisecond
	log.Printf("log-viewer started (delay=%dms between lines, Ctrl+C to exit)", delayMs)

	// Горутина: приём и вывод логов
	go func() {
		for {
			msg, err := consumer.Receive(context.Background())
			if err != nil {
				log.Printf("receive: %v", err)
				time.Sleep(time.Second)
				continue
			}

			// Парсим JSON (Service, Level, Message, Timestamp)
			var entry logcollector.LogEntry
			if err := json.Unmarshal(msg.Payload(), &entry); err != nil {
				consumer.Ack(msg)
				continue
			}

			// Форматируем: дата, время, сервис, уровень, сообщение
			t := time.Unix(entry.Timestamp, 0)
			dateStr := t.Format("2006/01/02")
			timeStr := t.Format("15:04:05")
			code := colorCode(entry.Level)   // ANSI-цвет по уровню
			reset := "\033[0m"
			fmt.Printf("%s%s [%s] [%-16s] [%-8s] %s%s\n", code, dateStr, timeStr, entry.Service, entry.Level, entry.Message, reset)

			time.Sleep(delay) // задержка для читаемости
			consumer.Ack(msg)
		}
	}()

	// Ожидаем Ctrl+C
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("log-viewer stopped")
}

// colorCode возвращает ANSI-код цвета фона по уровню лога.
func colorCode(level string) string {
	switch level {
	case "error":
		return "\033[41;97m"    // красный фон (критичные ошибки)
	case "warning", "warn":
		return "\033[43;30m"    // жёлтый фон (предупреждения)
	default:
		return "\033[42;30m"    // зелёный фон (info)
	}
}
