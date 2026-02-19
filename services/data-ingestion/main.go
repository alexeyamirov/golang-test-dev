// Сервис data-ingestion: читает TR181 данные из Pulsar и сохраняет метрики в PostgreSQL.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pulsarclient "github.com/apache/pulsar-client-go/pulsar"
	"golang-test-dev/pkg/database"
	"golang-test-dev/pkg/logcollector"
	"golang-test-dev/pkg/pulsar"
)

func main() {
	// Загружаем конфигурацию из env
	cfg := LoadConfig()

	// Подключаемся к PostgreSQL
	db, err := database.NewPostgresDB(cfg.PostgresConnStr)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	// Создаём таблицы (если ещё не созданы)
	if err := db.InitSchema(context.Background()); err != nil {
		log.Printf("schema: %v", err)
	}

	// Подключаемся к Pulsar
	client, err := pulsar.NewClient(cfg.PulsarURL)
	if err != nil {
		log.Fatalf("pulsar: %v", err)
	}
	defer client.Close()

	// Лог-producer создаём первым (до consumer) — иначе может не подключиться
	logColl := logcollector.NewFromClient(client, "data-ingestion", false)

	// Подписываемся на топик с данными устройств
	consumer, err := client.Subscribe(pulsarclient.ConsumerOptions{
		Topic:            pulsar.TopicTR181Data,
		SubscriptionName: "data-ingestion-sub",
		Type:             pulsarclient.Shared,
	})
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	storage := NewMetricStorage(db)
	handler := NewMessageHandler(storage, consumer, logColl)

	// Горутина: бесконечный цикл приёма и обработки сообщений
	go func() {
		for {
			msg, err := consumer.Receive(context.Background())
			if err != nil {
				log.Printf("receive: %v", err)
				time.Sleep(time.Second)
				continue
			}
			handler.Handle(context.Background(), msg)
		}
	}()

	log.Println("data-ingestion started")

	// Ожидаем сигнал завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down")
}
