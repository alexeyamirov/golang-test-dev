# Скрипт для быстрой остановки всех сервисов TR181 Cloud Platform

Write-Host "=== Stopping TR181 Cloud Platform ===" -ForegroundColor Yellow
Write-Host ""

# Остановка приложений Go
Write-Host "1. Stopping Go applications..." -ForegroundColor Cyan

$processes = @("api-gateway", "data-ingestion", "alert-processor", "simulator")
$stopped = 0

foreach ($proc in $processes) {
    $running = Get-Process -Name $proc -ErrorAction SilentlyContinue
    if ($running) {
        Write-Host "   Stopping $proc..." -ForegroundColor Gray
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
        $stopped++
    }
}

if ($stopped -gt 0) {
    Write-Host "   Stopped $stopped application(s)" -ForegroundColor Green
} else {
    Write-Host "   No running applications found" -ForegroundColor Gray
}

# Остановка Docker контейнеров
Write-Host ""
Write-Host "2. Stopping Docker containers..." -ForegroundColor Cyan

try {
    docker-compose down
    if ($LASTEXITCODE -eq 0) {
        Write-Host "   Docker containers stopped" -ForegroundColor Green
    } else {
        Write-Host "   Some containers may not have stopped" -ForegroundColor Yellow
    }
} catch {
    Write-Host "   Error stopping containers: $_" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== All services stopped ===" -ForegroundColor Green
Write-Host ""
