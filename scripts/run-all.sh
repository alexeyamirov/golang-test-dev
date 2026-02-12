#!/bin/bash

# Скрипт для запуска всех сервисов

echo "Starting TR181 Cloud Platform..."

# Проверка наличия Docker
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker first."
    exit 1
fi

# Запуск инфраструктуры
echo "Starting infrastructure (PostgreSQL, Redis, NATS)..."
docker-compose up -d

# Ожидание готовности сервисов
echo "Waiting for services to be ready..."
sleep 10

# Проверка готовности PostgreSQL
until docker exec tr181-postgres pg_isready -U postgres > /dev/null 2>&1; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done

# Проверка готовности Redis
until docker exec tr181-redis redis-cli ping > /dev/null 2>&1; do
    echo "Waiting for Redis..."
    sleep 2
done

echo "Infrastructure is ready!"

# Сборка приложений
echo "Building applications..."
make build

# Запуск сервисов в фоне
echo "Starting services..."

# API Gateway
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
REDIS_ADDR="localhost:6379" \
PORT=8080 \
./bin/api-gateway > logs/api-gateway.log 2>&1 &
API_GATEWAY_PID=$!

# Data Ingestion
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
PORT=8081 \
./bin/data-ingestion > logs/data-ingestion.log 2>&1 &
INGESTION_PID=$!

# Alert Processor
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
NATS_URL="nats://localhost:4222" \
./bin/alert-processor > logs/alert-processor.log 2>&1 &
PROCESSOR_PID=$!

# Небольшая задержка перед запуском симулятора
sleep 5

# Simulator
INGESTION_URL="http://localhost:8081/ingest" \
./bin/simulator > logs/simulator.log 2>&1 &
SIMULATOR_PID=$!

echo "All services started!"
echo "API Gateway PID: $API_GATEWAY_PID"
echo "Data Ingestion PID: $INGESTION_PID"
echo "Alert Processor PID: $PROCESSOR_PID"
echo "Simulator PID: $SIMULATOR_PID"
echo ""
echo "To stop all services, run: ./scripts/stop-all.sh"
echo "Or press Ctrl+C and run: kill $API_GATEWAY_PID $INGESTION_PID $PROCESSOR_PID $SIMULATOR_PID"

# Ожидание сигнала для остановки
trap "echo 'Stopping services...'; kill $API_GATEWAY_PID $INGESTION_PID $PROCESSOR_PID $SIMULATOR_PID 2>/dev/null; exit" INT TERM

wait
