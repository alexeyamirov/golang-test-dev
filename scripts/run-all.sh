#!/bin/bash
# Запуск всех сервисов одной командой (Linux и macOS)
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

# Очистка Pulsar volume (избегаем Bookie/ledger ошибок)
echo "1. Resetting Pulsar (clean volume)..."
docker-compose down 2>/dev/null || true
for vol in $(docker volume ls -q 2>/dev/null | grep pulsar_data || true); do
    docker volume rm "$vol" 2>/dev/null && echo "   Pulsar volume removed" || true
    break
done

# Запуск Docker
echo ""
echo "2. Starting Docker (PostgreSQL, Redis, Pulsar)..."
docker-compose up -d

# Ожидание PostgreSQL
echo "3. Waiting for PostgreSQL..."
for i in $(seq 1 15); do
    if docker exec tr181-postgres pg_isready -U postgres &> /dev/null; then
        echo "   PostgreSQL is ready"
        break
    fi
    [ $i -eq 15 ] && { echo "   PostgreSQL timeout"; exit 1; }
    sleep 2
done

# Ожидание Redis
echo "   Checking Redis..."
docker exec tr181-redis redis-cli ping &> /dev/null && echo "   Redis is ready" || echo "   Redis check skipped"

# Ожидание Pulsar (60 сек)
echo "   Waiting for Pulsar (60s)..."
sleep 60

# Сборка
echo ""
echo "4. Building applications..."
chmod +x ./build.sh 2>/dev/null || true
./build.sh

# Папка для логов
mkdir -p logs

# Переменные окружения
export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
export PULSAR_URL="pulsar://localhost:6650"
export REDIS_ADDR="localhost:6379"

# Запуск сервисов в фоне
echo ""
echo "5. Starting services..."

echo "   API Gateway..."
PORT=8080 GRPC_PORT=9090 ./bin/api-gateway > logs/api-gateway.log 2>&1 &
API_GATEWAY_PID=$!

echo "   Data Ingestion..."
./bin/data-ingestion > logs/data-ingestion.log 2>&1 &
INGESTION_PID=$!

echo "   Alert Processor..."
./bin/alert-processor > logs/alert-processor.log 2>&1 &
PROCESSOR_PID=$!

sleep 2

echo "   Simulator..."
./bin/simulator > logs/simulator.log 2>&1 &
SIMULATOR_PID=$!

# PID для stop-all.sh
echo "$API_GATEWAY_PID" > logs/api-gateway.pid
echo "$INGESTION_PID" > logs/data-ingestion.pid
echo "$PROCESSOR_PID" > logs/alert-processor.pid
echo "$SIMULATOR_PID" > logs/simulator.pid

echo ""
echo "=== All services started! ==="
echo "  API Gateway:    http://localhost:8080   gRPC: localhost:9090"
echo "  Data Ingestion: consuming from Pulsar"
echo "  Alert Processor: consuming from Pulsar"
echo "  Simulator:      publishing to Pulsar"
echo ""
echo "Logs: logs/*.log"
echo ""
echo "Optional log-viewer (in separate terminal):"
echo "  export PULSAR_URL=\"pulsar://localhost:6650\""
echo "  ./bin/log-viewer"
echo ""
echo "To stop: ./scripts/stop-all.sh"
echo ""

trap "echo 'Stopping...'; kill $API_GATEWAY_PID $INGESTION_PID $PROCESSOR_PID $SIMULATOR_PID 2>/dev/null; exit" INT TERM

wait
