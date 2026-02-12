#!/bin/bash
# Запуск всех сервисов в фоне (Linux и macOS)
# Использование: ./scripts/run-all.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

echo "=== Starting TR181 Cloud Platform ==="

# Проверка Docker
if ! command -v docker &> /dev/null; then
    echo "Docker is not installed. Please install Docker first."
    exit 1
fi

# Запуск инфраструктуры
echo "Starting Docker containers (PostgreSQL, Redis, Pulsar)..."
docker-compose up -d

# Ожидание готовности
echo "Waiting for services to be ready..."
sleep 15

# Проверка PostgreSQL
until docker exec tr181-postgres pg_isready -U postgres &> /dev/null; do
    echo "Waiting for PostgreSQL..."
    sleep 2
done
echo "PostgreSQL is ready"

# Проверка Redis
until docker exec tr181-redis redis-cli ping &> /dev/null; do
    echo "Waiting for Redis..."
    sleep 2
done
echo "Redis is ready"

# Ожидание Pulsar (дольше запускается)
echo "Waiting for Pulsar (45s)..."
sleep 45

# Сборка приложений
echo "Building applications..."
if [ -f "./build.sh" ]; then
    chmod +x ./build.sh
    ./build.sh
else
    make build 2>/dev/null || {
        go build -o bin/api-gateway ./services/api-gateway
        go build -o bin/data-ingestion ./services/data-ingestion
        go build -o bin/alert-processor ./services/alert-processor
        go build -o bin/simulator ./simulator
    }
fi

# Создаём папку для логов
mkdir -p logs

# Переменные окружения
export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
export PULSAR_URL="pulsar://localhost:6650"
export REDIS_ADDR="localhost:6379"

# Запуск API Gateway
echo "Starting API Gateway..."
PORT=8080 GRPC_PORT=9090 ./bin/api-gateway > logs/api-gateway.log 2>&1 &
API_GATEWAY_PID=$!

# Запуск Data Ingestion
echo "Starting Data Ingestion..."
./bin/data-ingestion > logs/data-ingestion.log 2>&1 &
INGESTION_PID=$!

# Запуск Alert Processor
echo "Starting Alert Processor..."
./bin/alert-processor > logs/alert-processor.log 2>&1 &
PROCESSOR_PID=$!

sleep 3

# Запуск Simulator
echo "Starting Simulator..."
./bin/simulator > logs/simulator.log 2>&1 &
SIMULATOR_PID=$!

# Сохраняем PID для stop-all.sh
echo "$API_GATEWAY_PID" > logs/api-gateway.pid
echo "$INGESTION_PID" > logs/data-ingestion.pid
echo "$PROCESSOR_PID" > logs/alert-processor.pid
echo "$SIMULATOR_PID" > logs/simulator.pid

echo ""
echo "=== All services started! ==="
echo "  API Gateway:    http://localhost:8080 (HTTP), localhost:9090 (gRPC)"
echo "  Data Ingestion: consuming from Pulsar"
echo "  Alert Processor: consuming from Pulsar"
echo "  Simulator:      publishing to Pulsar"
echo ""
echo "Logs: logs/*.log"
echo ""
echo "To stop: ./scripts/stop-all.sh"
echo ""

# Ожидание сигнала для остановки
trap "echo 'Stopping services...'; kill $API_GATEWAY_PID $INGESTION_PID $PROCESSOR_PID $SIMULATOR_PID 2>/dev/null; exit" INT TERM

wait
