# Быстрый старт

## Шаг 1: Запуск инфраструктуры

```bash
docker-compose up -d
```

Подождите 10-15 секунд, пока все сервисы запустятся.

## Шаг 2: Сборка приложений

```bash
go mod download
make build
```

## Шаг 3: Запуск сервисов

Откройте 4 терминала и выполните в каждом:

**Терминал 1 - API Gateway:**
```bash
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" REDIS_ADDR="localhost:6379" PORT=8080 ./bin/api-gateway
```

**Терминал 2 - Data Ingestion:**
```bash
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" NATS_URL="nats://localhost:4222" PORT=8081 ./bin/data-ingestion
```

**Терминал 3 - Alert Processor:**
```bash
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" NATS_URL="nats://localhost:4222" ./bin/alert-processor
```

**Терминал 4 - Simulator:**
```bash
INGESTION_URL="http://localhost:8081/ingest" ./bin/simulator
```

## Шаг 4: Тестирование

Подождите 30-60 секунд, чтобы симулятор отправил данные, затем:

```bash
# Проверка метрик
curl "http://localhost:8080/api/v1/metric/cpu-usage?serial-number=DEV-00000001&from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z"

# Проверка алертов
curl "http://localhost:8080/api/v1/alert/high-cpu-usage?serial-number=DEV-00000001&from=2024-01-01T00:00:00Z&to=2024-01-31T23:59:59Z"
```

## Остановка

Нажмите `Ctrl+C` в каждом терминале, затем:

```bash
docker-compose down
```
