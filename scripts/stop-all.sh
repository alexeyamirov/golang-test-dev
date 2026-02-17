#!/bin/bash
# Остановка всех сервисов TR181 Cloud Platform (Linux и macOS)
# Использование: ./scripts/stop-all.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
LOGS_DIR="$PROJECT_DIR/logs"

echo "Stopping TR181 Cloud Platform..."

# Остановка по PID файлам (если есть)
if [ -d "$LOGS_DIR" ]; then
    for pidfile in api-gateway data-ingestion alert-processor simulator; do
        if [ -f "$LOGS_DIR/${pidfile}.pid" ]; then
            pid=$(cat "$LOGS_DIR/${pidfile}.pid" 2>/dev/null)
            if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
                echo "  Stopping $pidfile (PID $pid)..."
                kill "$pid" 2>/dev/null || true
            fi
        fi
    done
fi

# Остановка по имени процесса (на случай если PID файлы не созданы)
pkill -f "bin/api-gateway" 2>/dev/null || true
pkill -f "bin/data-ingestion" 2>/dev/null || true
pkill -f "bin/alert-processor" 2>/dev/null || true
pkill -f "bin/simulator" 2>/dev/null || true
pkill -f "bin/log-viewer" 2>/dev/null || true

# Остановка Docker контейнеров
echo "Stopping Docker containers..."
docker-compose down

echo "All services stopped."
