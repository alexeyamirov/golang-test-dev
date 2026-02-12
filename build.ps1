# Скрипт сборки проекта (аналог make build)
# Использование: .\build.ps1

Write-Host "Building TR181 Cloud Platform..." -ForegroundColor Cyan

go mod download
go build -o bin/api-gateway.exe ./services/api-gateway
go build -o bin/data-ingestion.exe ./services/data-ingestion
go build -o bin/alert-processor.exe ./services/alert-processor
go build -o bin/simulator.exe ./simulator

Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Binaries in bin/ folder" -ForegroundColor Gray
