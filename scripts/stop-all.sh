#!/bin/bash

echo "Stopping all services..."

# Остановка всех процессов
pkill -f api-gateway
pkill -f data-ingestion
pkill -f alert-processor
pkill -f simulator

# Остановка Docker контейнеров
docker-compose down

echo "All services stopped."
