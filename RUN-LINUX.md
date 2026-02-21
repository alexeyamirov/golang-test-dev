# Запуск TR181 Cloud Platform на Linux и macOS

## Требования

- **Go** 1.21+ 
- **Docker** и **Docker Compose**
- **Git** (опционально)

---

## Быстрый старт

### Вариант 1: Пошагово с инструкциями

```bash
# Сделать скрипты исполняемыми (один раз)
chmod +x build.sh start-all.sh stop-all.sh test-api.sh scripts/*.sh

# Запустить окружение и собрать проект
./start-all.sh
```

После выполнения следуйте выведенным инструкциям и запустите 4 терминала с нужными переменными окружения.

### Вариант 2: Одна команда — все в фоне

```bash
chmod +x scripts/run-all.sh scripts/stop-all.sh

# Запуск: Docker + сборка + все 4 сервиса
./scripts/run-all.sh

# Остановка
./scripts/stop-all.sh
```

Скрипт: очищает Pulsar volume, поднимает Docker, ждет 60 сек, собирает приложения и запускает сервисы. Логи в `logs/*.log`.

Опционально log-viewer в отдельном терминале:
```bash
export PULSAR_URL="pulsar://localhost:6650"
./bin/log-viewer
```

### Вариант 3: Ручной запуск

```bash
# 1. Инфраструктура
docker-compose up -d
sleep 45   # Ожидание Pulsar

# 2. Сборка
./build.sh

# 3. Терминалы (по одному на сервис)
# Терминал 1
export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
export REDIS_ADDR="localhost:6379" PULSAR_URL="pulsar://localhost:6650"
export PORT=8080 GRPC_PORT=9090
./bin/api-gateway

# Терминал 2
export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
export PULSAR_URL="pulsar://localhost:6650"
./bin/data-ingestion

# Терминал 3
export POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
export PULSAR_URL="pulsar://localhost:6650"
./bin/alert-processor

# Терминал 4
export PULSAR_URL="pulsar://localhost:6650"
./bin/simulator
```

---

## Скрипты

| Скрипт | Описание |
|--------|----------|
| `build.sh` | Сборка всех приложений |
| `start-all.sh` | Docker + сборка + инструкции для запуска |
| `scripts/run-all.sh` | Полный запуск в фоне |
| `scripts/stop-all.sh` | Остановка сервисов и Docker |
| `stop-all.sh` | Обертка над scripts/stop-all.sh |
| `test-api.sh` | Проверка HTTP и gRPC API |

---

## Проверка работы

```bash
# Health check
curl http://localhost:8080/health

# Тест API
./test-api.sh

# gRPC (если установлен grpcurl)
grpcurl -plaintext -d '{"metric_type":"cpu-usage","serial_number":"DEV-00000001","from":0,"to":0}' \
  localhost:9090 tr181.api.TR181Api/GetMetric
```

---

## Остановка

```bash
./scripts/stop-all.sh
```

Или:

```bash
./stop-all.sh
```

---

## Полезные команды

```bash
# Логи (при run-all.sh)
tail -f logs/api-gateway.log
tail -f logs/simulator.log
tail -f logs/data-ingestion.log
tail -f logs/alert-processor.log

# Статус Docker
docker ps

# Пересборка
./build.sh
```
