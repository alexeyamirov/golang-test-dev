#!/bin/bash
# Скрипт для быстрого запуска TR181 Cloud Platform (Linux и macOS)
# Использование: ./start-all.sh

set -e

echo "=== TR181 Cloud Platform Startup Script ==="
echo ""

# 1. Проверка Docker
echo "1. Checking Docker..."
if ! command -v docker &> /dev/null; then
    echo "   Docker is not installed. Please install Docker first."
    exit 1
fi
if ! docker ps &> /dev/null; then
    echo "   Docker is not running. Please start Docker first."
    exit 1
fi
echo "   Docker is running"

# 2. Запуск Docker контейнеров
echo ""
echo "2. Starting Docker containers..."
docker-compose up -d
if [ $? -ne 0 ]; then
    echo "   Failed to start containers"
    exit 1
fi
echo "   Containers started"

# 3. Ожидание готовности сервисов
echo ""
echo "3. Waiting for services to be ready..."
sleep 10

# Проверка PostgreSQL
echo "   Checking PostgreSQL..."
for i in $(seq 1 10); do
    if docker exec tr181-postgres pg_isready -U postgres &> /dev/null; then
        echo "   PostgreSQL is ready"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "   PostgreSQL is not ready"
        exit 1
    fi
    sleep 2
done

# Проверка Redis
echo "   Checking Redis..."
if docker exec tr181-redis redis-cli ping &> /dev/null; then
    echo "   Redis is ready"
else
    echo "   Redis check failed (may still work)"
fi

# Ожидание Pulsar (требует больше времени)
echo "   Waiting for Pulsar (45s)..."
sleep 45

# 4. Сборка приложений
echo ""
echo "4. Building applications..."
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
echo "   All applications built"

# 5. Инструкции
echo ""
echo "=== Setup Complete ==="
echo ""
echo "Now start the services in separate terminals:"
echo ""
echo "Terminal 1 - API Gateway (HTTP + gRPC):"
echo '  export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
echo '  export REDIS_ADDR="localhost:6379"'
echo '  export PORT="8080"'
echo '  export GRPC_PORT="9090"'
echo '  ./bin/api-gateway'
echo ""
echo "Terminal 2 - Data Ingestion:"
echo '  export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
echo '  export PULSAR_URL="pulsar://localhost:6650"'
echo '  ./bin/data-ingestion'
echo ""
echo "Terminal 3 - Alert Processor:"
echo '  export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
echo '  export PULSAR_URL="pulsar://localhost:6650"'
echo '  ./bin/alert-processor'
echo ""
echo "Terminal 4 - Simulator:"
echo '  export PULSAR_URL="pulsar://localhost:6650"'
echo '  ./bin/simulator'
echo ""
echo "Or run all in background: ./scripts/run-all.sh"
echo "To stop: ./scripts/stop-all.sh"
echo ""
