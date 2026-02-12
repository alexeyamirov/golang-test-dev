# TR181 Cloud Platform

Тестовая платформа для обработки TR181 данных от роутеров с поддержкой метрик и алертов.

## Архитектура

Проект состоит из следующих компонентов:

1. **API Gateway** (`services/api-gateway`) - HTTP API для запроса метрик и алертов
2. **Data Ingestion Service** (`services/data-ingestion`) - сервис приема TR181 данных от симулятора
3. **Alert Processor** (`services/alert-processor`) - фоновый процессор для обработки алертов
4. **Simulator** (`simulator`) - симулятор 20K устройств, отправляющих TR181 данные

### Инфраструктура

- **PostgreSQL + TimescaleDB** - хранение временных рядов метрик и алертов
- **Redis** - кэширование запросов для улучшения производительности
- **Apache Pulsar** - message broker для TR181 данных и алертов

## Поддерживаемые метрики

- `cpu-usage` - использование CPU (0-100%)
- `memory-usage` - использование памяти (0-100%)
- `cpu-temperature` - температура CPU (°C)
- `board-temperature` - температура платы (°C)
- `radio-temperature` - температура радио модуля (°C)
- `wifi-2ghz-signal` - сила сигнала WiFi 2.4 GHz (dBm)
- `wifi-5ghz-signal` - сила сигнала WiFi 5 GHz (dBm)
- `wifi-6ghz-signal` - сила сигнала WiFi 6 GHz (dBm)
- `ethernet-bytes-sent` - отправлено байт по Ethernet
- `ethernet-bytes-received` - получено байт по Ethernet
- `uptime` - время работы устройства (секунды)

## Поддерживаемые алерты

- `high-cpu-usage` - CPU usage > 60%
- `low-wifi` - WiFi signal strength < -100 dBm

## API Endpoints

### Метрики

```
GET /api/v1/metric/{metric-type}?serial-number={serial-number}&from={from}&to={to}
```

Пример:
```bash
curl "http://localhost:8080/api/v1/metric/cpu-usage?serial-number=DEV-00000001&from=2024-01-01T00:00:00Z&to=2024-01-01T23:59:59Z"
```

Ответ:
```json
[
  {
    "value": 45,
    "time": 1704067200
  },
  {
    "value": 47,
    "time": 1704067230
  }
]
```

### Алерты

```
GET /api/v1/alert/{alert-type}?serial-number={serial-number}&from={from}&to={to}
```

Пример:
```bash
curl "http://localhost:8080/api/v1/alert/high-cpu-usage?serial-number=DEV-00000001&from=2024-01-01T00:00:00Z&to=2024-01-01T23:59:59Z"
```

Ответ:
```json
{
  "value": 72,
  "count": 15
}
```

## Установка и запуск

### Требования

- Go 1.21+
- Docker и Docker Compose
- Make (опционально, на Windows используйте `.\build.ps1`)

### Windows

```powershell
.\start-all.ps1    # Docker + сборка + инструкции
.\build.ps1        # Только сборка
.\test-api.ps1     # Тест API
.\stop-all.ps1     # Остановка
```

См. [START.md](START.md) — пошаговая инструкция.

### Linux и macOS

```bash
chmod +x build.sh start-all.sh stop-all.sh test-api.sh scripts/*.sh

./start-all.sh         # Docker + сборка + инструкции
./scripts/run-all.sh   # Полный запуск в фоне
./test-api.sh          # Тест API
./scripts/stop-all.sh  # Остановка
```

См. [RUN-LINUX.md](RUN-LINUX.md) — подробная инструкция.

### Шаги (универсально)

1. **Запуск инфраструктуры:**
```bash
make up
# или
docker-compose up -d
```

2. **Сборка всех сервисов:**
```bash
make build
```

3. **Запуск сервисов (в отдельных терминалах):**

```bash
# Terminal 1: API Gateway
make run-api

# Terminal 2: Data Ingestion
make run-ingestion

# Terminal 3: Alert Processor
make run-processor

# Terminal 4: Simulator
make run-simulator
```

Или запустить все через переменные окружения:

```bash
# Terminal 1
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
REDIS_ADDR="localhost:6379" \
PORT=8080 \
./bin/api-gateway

# Terminal 2
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
PULSAR_URL="pulsar://localhost:6650" \
./bin/data-ingestion

# Terminal 3
POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable" \
PULSAR_URL="pulsar://localhost:6650" \
./bin/alert-processor

# Terminal 4
PULSAR_URL="pulsar://localhost:6650" \
./bin/simulator
```

## Переменные окружения

### API Gateway
- `PORT` - порт для HTTP (по умолчанию: 8080)
- `GRPC_PORT` - порт для gRPC (по умолчанию: 9090)
- `POSTGRES_CONN_STR` - строка подключения к PostgreSQL
- `REDIS_ADDR` - адрес Redis сервера

### Data Ingestion
- `POSTGRES_CONN_STR` - строка подключения к PostgreSQL
- `PULSAR_URL` - URL Apache Pulsar (по умолчанию: pulsar://localhost:6650)

### Alert Processor
- `POSTGRES_CONN_STR` - строка подключения к PostgreSQL
- `PULSAR_URL` - URL Apache Pulsar

### Simulator
- `PULSAR_URL` - URL Apache Pulsar

## Производительность

- **Response time**: < 1 секунда (благодаря кэшированию в Redis)
- **Масштабируемость**: Каждый сервис создает CPU нагрузку для автоскейлинга
- **Нагрузка**: Симулятор генерирует данные от 20,000 устройств каждые 30 секунд

## Технические решения

- **Транспорт**: HTTP REST и gRPC для API, Apache Pulsar для межсервисного взаимодействия
- **Протокол TR181**: JSON формат с поддержкой customer extensions
- **База данных**: PostgreSQL с TimescaleDB для временных рядов, Redis для кэширования
- **Обработка алертов**: Асинхронная через Apache Pulsar с возможной задержкой

## Структура проекта

```
.
├── pkg/
│   ├── tr181/          # TR181 модель данных
│   └── database/       # Работа с БД (PostgreSQL, Redis)
├── services/
│   ├── api-gateway/    # API Gateway сервис
│   ├── data-ingestion/ # Сервис приема данных
│   └── alert-processor/# Процессор алертов
├── simulator/          # Симулятор устройств
├── docker-compose.yml  # Docker инфраструктура
├── Makefile           # Команды для сборки и запуска
└── README.md          # Документация
```

## Лицензия

Тестовый проект для демонстрации архитектуры обработки TR181 данных.
