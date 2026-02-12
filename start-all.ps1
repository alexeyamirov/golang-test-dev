# Скрипт для быстрого запуска всех сервисов TR181 Cloud Platform
# Запустите этот скрипт от имени администратора

Write-Host "=== TR181 Cloud Platform Startup Script ===" -ForegroundColor Green
Write-Host ""

# Проверка Docker
Write-Host "1. Checking Docker..." -ForegroundColor Cyan
try {
    docker ps | Out-Null
    Write-Host "   Docker is running" -ForegroundColor Green
} catch {
    Write-Host "   Docker is not running. Please start Docker Desktop first." -ForegroundColor Red
    exit 1
}

# Запуск Docker контейнеров
Write-Host ""
Write-Host "2. Starting Docker containers..." -ForegroundColor Cyan
docker-compose up -d
if ($LASTEXITCODE -ne 0) {
    Write-Host "   Failed to start containers" -ForegroundColor Red
    exit 1
}
Write-Host "   Containers started" -ForegroundColor Green

# Ожидание готовности сервисов
Write-Host ""
Write-Host "3. Waiting for services to be ready..." -ForegroundColor Cyan
Start-Sleep -Seconds 10

# Проверка готовности PostgreSQL
Write-Host "   Checking PostgreSQL..." -ForegroundColor Gray
$maxAttempts = 10
$attempt = 0
while ($attempt -lt $maxAttempts) {
    try {
        docker exec tr181-postgres pg_isready -U postgres | Out-Null
        Write-Host "   PostgreSQL is ready" -ForegroundColor Green
        break
    } catch {
        $attempt++
        if ($attempt -ge $maxAttempts) {
            Write-Host "   PostgreSQL is not ready" -ForegroundColor Red
            exit 1
        }
        Start-Sleep -Seconds 2
    }
}

# Проверка готовности Redis
Write-Host "   Checking Redis..." -ForegroundColor Gray
try {
    docker exec tr181-redis redis-cli ping | Out-Null
    Write-Host "   Redis is ready" -ForegroundColor Green
} catch {
    Write-Host "   Redis check failed (may still work)" -ForegroundColor Yellow
}

# Ожидание Pulsar (может потребоваться больше времени)
Write-Host "   Waiting for Pulsar (30s)..." -ForegroundColor Gray
Start-Sleep -Seconds 30

# Сборка приложений
Write-Host ""
Write-Host "4. Building applications..." -ForegroundColor Cyan
Write-Host "   Building API Gateway..." -ForegroundColor Gray
go build -o bin/api-gateway.exe ./services/api-gateway
if ($LASTEXITCODE -ne 0) {
    Write-Host "   Failed to build API Gateway" -ForegroundColor Red
    exit 1
}

Write-Host "   Building Data Ingestion..." -ForegroundColor Gray
go build -o bin/data-ingestion.exe ./services/data-ingestion
if ($LASTEXITCODE -ne 0) {
    Write-Host "   Failed to build Data Ingestion" -ForegroundColor Red
    exit 1
}

Write-Host "   Building Alert Processor..." -ForegroundColor Gray
go build -o bin/alert-processor.exe ./services/alert-processor
if ($LASTEXITCODE -ne 0) {
    Write-Host "   Failed to build Alert Processor" -ForegroundColor Red
    exit 1
}

Write-Host "   Building Simulator..." -ForegroundColor Gray
go build -o bin/simulator.exe ./simulator
if ($LASTEXITCODE -ne 0) {
    Write-Host "   Failed to build Simulator" -ForegroundColor Red
    exit 1
}

Write-Host "   All applications built" -ForegroundColor Green

# Инструкции по запуску
Write-Host ""
Write-Host "=== Setup Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Now start the services in separate terminals:" -ForegroundColor Yellow
Write-Host ""
Write-Host "Terminal 1 - API Gateway (HTTP + gRPC):" -ForegroundColor Cyan
Write-Host '  $env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
Write-Host '  $env:REDIS_ADDR="localhost:6379"'
Write-Host '  $env:PORT="8080"'
Write-Host '  $env:GRPC_PORT="9090"'
Write-Host '  .\bin\api-gateway.exe'
Write-Host ""
Write-Host "Terminal 2 - Data Ingestion:" -ForegroundColor Cyan
Write-Host '  $env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
Write-Host '  $env:PULSAR_URL="pulsar://localhost:6650"'
Write-Host '  $env:PORT="8081"'
Write-Host '  .\bin\data-ingestion.exe'
Write-Host ""
Write-Host "Terminal 3 - Alert Processor:" -ForegroundColor Cyan
Write-Host '  $env:POSTGRES_CONN_STR="postgres://postgres:postgres@localhost:5432/tr181?sslmode=disable"'
Write-Host '  $env:PULSAR_URL="pulsar://localhost:6650"'
Write-Host '  .\bin\alert-processor.exe'
Write-Host ""
Write-Host "Terminal 4 - Simulator:" -ForegroundColor Cyan
Write-Host '  $env:PULSAR_URL="pulsar://localhost:6650"'
Write-Host '  .\bin\simulator.exe'
Write-Host ""
