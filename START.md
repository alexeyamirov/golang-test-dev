# Пошаговая инструкция запуска TR181 Cloud Platform

## После перезагрузки компьютера

### Шаг 1: Запуск инфраструктуры (Docker)

1. **Откройте PowerShell от имени администратора:**
   - Нажмите `Win + X`
   - Выберите "Windows PowerShell (Admin)" или "Terminal (Admin)"

2. **Перейдите в директорию проекта:**
   ```powershell
   cd E:\works\education\go\golang-test-dev
   ```

3. **Запустите Docker контейнеры:**
   ```powershell
   docker-compose up -d
   ```

4. **Проверьте, что контейнеры запущены:**
   ```powershell
   docker ps
   ```
   
   Должны быть видны 4 контейнера:
   - `tr181-postgres`
   - `tr181-redis`
   - `tr181-pulsar` (Apache Pulsar)

5. **Подождите 45–60 секунд**, пока Pulsar полностью запустится (ему нужно больше времени, чем PostgreSQL и Redis)

---

### Шаг 2: Сборка приложений

1. **Откройте обычный PowerShell** (не обязательно админский)

2. **Перейдите в директорию проекта:**
   ```powershell
   cd E:\works\education\go\golang-test-dev
   ```

3. **Соберите все приложения:**
   ```powershell
   go build -o bin/api-gateway.exe ./services/api-gateway
   go build -o bin/data-ingestion.exe ./services/data-ingestion
   go build -o bin/alert-processor.exe ./services/alert-processor
   go build -o bin/simulator.exe ./simulator
   ```

   Или используйте скрипт (на Windows `make` обычно не установлен):
   ```powershell
   .\build.ps1
   ```

---

### Шаг 3: Запуск сервисов

Откройте **4 отдельных терминала** (PowerShell) и выполните в каждом:

#### Терминал 1 - API Gateway (HTTP + gRPC):
```powershell
cd E:\works\education\go\golang-test-dev
$env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
$env:REDIS_ADDR="localhost:6379"
$env:PORT="8080"
$env:GRPC_PORT="9090"
.\bin\api-gateway.exe
```

#### Терминал 2 - Data Ingestion:
```powershell
cd E:\works\education\go\golang-test-dev
$env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
$env:PULSAR_URL="pulsar://localhost:6650"
.\bin\data-ingestion.exe
```

#### Терминал 3 - Alert Processor:
```powershell
cd E:\works\education\go\golang-test-dev
$env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"
$env:PULSAR_URL="pulsar://localhost:6650"
.\bin\alert-processor.exe
```

#### Терминал 4 - Simulator:
```powershell
cd E:\works\education\go\golang-test-dev
$env:PULSAR_URL="pulsar://localhost:6650"
.\bin\simulator.exe
```

---

### Шаг 4: Проверка работы

1. **Проверьте health check API Gateway:**
   ```powershell
   # API Gateway (HTTP)
   Invoke-WebRequest -Uri http://localhost:8080/health -UseBasicParsing
   
   # API Gateway (gRPC) - если установлен grpcurl:
   grpcurl -plaintext localhost:9090 list
   ```

2. **Подождите 30-60 секунд**, чтобы симулятор отправил данные

3. **Проверьте метрики:**
   ```powershell
   $url = "http://localhost:8080/api/v1/metric/cpu-usage?serial-number=DEV-00000001&from=2024-01-01T00:00:00Z&to=2025-12-31T23:59:59Z"
   Invoke-WebRequest -Uri $url -UseBasicParsing | Select-Object -ExpandProperty Content
   ```

4. **Или запустите тестовый скрипт:**
   ```powershell
   .\test-api.ps1
   ```

---

## Остановка всех сервисов

### Быстрый способ (рекомендуется):

```powershell
.\stop-all.ps1
```

### Ручной способ:

1. **Остановите приложения:** Нажмите `Ctrl+C` в каждом терминале с запущенными сервисами

2. **Остановите Docker контейнеры:**
   ```powershell
   docker-compose down
   ```

Или остановите все процессы одной командой:
```powershell
Stop-Process -Name api-gateway,data-ingestion,alert-processor,simulator -Force -ErrorAction SilentlyContinue
docker-compose down
```

---

## Быстрая проверка статуса

```powershell
# Проверка Docker контейнеров
docker ps

# Проверка API Gateway
Invoke-WebRequest -Uri http://localhost:8080/health -UseBasicParsing

# Проверка gRPC (если установлен grpcurl)
grpcurl -plaintext localhost:9090 list tr181.api.TR181Api
```

---

## Устранение проблем

### Если Docker не запускается:
- Убедитесь, что Docker Desktop запущен
- Запустите PowerShell от имени администратора

### Если порты заняты:
- Проверьте, не запущены ли сервисы: `netstat -ano | findstr :8080`
- Остановите старые процессы или измените порты в переменных окружения

### Если нет данных в API:
- Подождите 30–60 секунд после запуска симулятора
- Проверьте логи симулятора — должны быть сообщения "Batch published" или "Publishing batch"
- Проверьте логи Data Ingestion — "Processed device"
- Убедитесь, что Pulsar запущен: `docker ps` (контейнер tr181-pulsar)

---

## Порты сервисов

- **8080** - API Gateway (HTTP)
- **9090** - API Gateway (gRPC)
- **6650** - Apache Pulsar (бинарный протокол)
- **5432** - PostgreSQL
- **6379** - Redis

---

## Полезные команды

```powershell
# Пересборка всех приложений
make build

# Просмотр логов Docker контейнеров
docker-compose logs -f

# Остановка всех контейнеров
docker-compose down

# Остановка и удаление всех данных
docker-compose down -v
```
