#!/bin/bash
# Скрипт сборки проекта для Linux и macOS
# Использование: ./build.sh

set -e

echo "Building TR181 Cloud Platform..."

# Загружаем зависимости
go mod download

# Собираем все приложения (без .exe на Linux/macOS)
go build -o bin/api-gateway ./services/api-gateway
go build -o bin/data-ingestion ./services/data-ingestion
go build -o bin/alert-processor ./services/alert-processor
go build -o bin/simulator ./simulator

echo "Build complete!"
echo "Binaries in bin/ folder"
