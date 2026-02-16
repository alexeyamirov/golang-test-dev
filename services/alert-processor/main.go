// Сервис alert-processor: читает TR181 данные из Pulsar и сохраняет алерты в PostgreSQL.
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
	"golang-test-dev/pkg/pulsar"
)

func main() {
	postgresConnStr := os.Getenv("POSTGRES_CONN_STR")
	if postgresConnStr == "" {
		postgresConnStr = "postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
	}

	pulsarURL := os.Getenv("PULSAR_URL")
	if pulsarURL == "" {
		pulsarURL = "pulsar://localhost:6650"
	}

	db, err := database.NewPostgresDB(postgresConnStr)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(context.Background()); err != nil {
		log.Printf("schema: %v", err)
	}

	client, err := pulsar.NewClient(pulsarURL)
	if err != nil {
		log.Fatalf("pulsar: %v", err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsarclient.ConsumerOptions{
		Topic:            pulsar.TopicTR181Data,
		SubscriptionName: "alert-processor-sub",
		Type:             pulsarclient.Shared,
	})
	if err != nil {
		log.Fatalf("consumer: %v", err)
	}
	defer consumer.Close()

	storage := NewAlertStorage(db)
	handler := NewAlertHandler(storage, consumer)

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

	go cpuLoadLoop()

	log.Println("alert-processor started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down")
}

// cpuLoadLoop создаёт фоновую CPU нагрузку (для тестов автоскейлинга).
func cpuLoadLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		sum := 0
		for i := 0; i < 10000; i++ {
			sum += i * i
		}
		_ = sum
	}
}
