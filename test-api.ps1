# Скрипт для тестирования API TR181 Cloud Platform

Write-Host "=== Testing TR181 Cloud Platform API ===" -ForegroundColor Green
Write-Host ""

# Ждем немного, чтобы данные накопились
Write-Host "Waiting 5 seconds for data to accumulate..." -ForegroundColor Yellow
Start-Sleep -Seconds 5

# Тест 1: Health checks
Write-Host "1. Testing Health Checks:" -ForegroundColor Cyan
try {
    $health1 = Invoke-WebRequest -Uri "http://localhost:8080/health" -UseBasicParsing
    Write-Host "   ✓ API Gateway (HTTP): $($health1.Content)" -ForegroundColor Green
} catch {
    Write-Host "   ✗ API Gateway health check failed: $_" -ForegroundColor Red
}

# gRPC test (if grpcurl installed)
if (Get-Command grpcurl -ErrorAction SilentlyContinue) {
    Write-Host "   Testing gRPC (port 9090)..." -ForegroundColor Gray
    try {
        grpcurl -plaintext -d '{"metric_type":"cpu-usage","serial_number":"DEV-00000001","from":0,"to":0}' localhost:9090 tr181.api.TR181Api/GetMetric 2>$null
        Write-Host "   ✓ API Gateway (gRPC): OK" -ForegroundColor Green
    } catch {
        Write-Host "   ⚠ gRPC test skipped" -ForegroundColor Yellow
    }
}

Write-Host ""

# Тест 2: Получение метрик
Write-Host "2. Testing Metrics API:" -ForegroundColor Cyan

$serialNumber = "DEV-00000001"
$from = "2024-01-01T00:00:00Z"
$to = "2025-12-31T23:59:59Z"

$metricTypes = @("cpu-usage", "memory-usage", "wifi-2ghz-signal", "cpu-temperature")

foreach ($metricType in $metricTypes) {
    try {
        $url = "http://localhost:8080/api/v1/metric/$metricType?serial-number=$serialNumber&from=$from&to=$to"
        $response = Invoke-WebRequest -Uri $url -UseBasicParsing
        $data = $response.Content | ConvertFrom-Json
        
        if ($data.Count -gt 0) {
            Write-Host "   ✓ $metricType : Found $($data.Count) data points" -ForegroundColor Green
            Write-Host "     Latest value: $($data[-1].value) at $($data[-1].time)" -ForegroundColor Gray
        } else {
            Write-Host "   ⚠ $metricType : No data yet (wait a bit longer)" -ForegroundColor Yellow
        }
    } catch {
        Write-Host "   ✗ $metricType : Error - $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""

# Тест 3: Получение алертов
Write-Host "3. Testing Alerts API:" -ForegroundColor Cyan

$alertTypes = @("high-cpu-usage", "low-wifi")

foreach ($alertType in $alertTypes) {
    try {
        $url = "http://localhost:8080/api/v1/alert/$alertType?serial-number=$serialNumber&from=$from&to=$to"
        $response = Invoke-WebRequest -Uri $url -UseBasicParsing
        $data = $response.Content | ConvertFrom-Json
        
        Write-Host "   ✓ $alertType :" -ForegroundColor Green
        Write-Host "     Average value: $($data.value)" -ForegroundColor Gray
        Write-Host "     Alert count: $($data.count)" -ForegroundColor Gray
    } catch {
        Write-Host "   ✗ $alertType : Error - $($_.Exception.Message)" -ForegroundColor Red
    }
}

Write-Host ""
Write-Host "=== Testing Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "Note: If you see no data, wait 30-60 seconds for the simulator to send data." -ForegroundColor Yellow
Write-Host "The simulator sends data every 30 seconds from 20,000 devices." -ForegroundColor Yellow
