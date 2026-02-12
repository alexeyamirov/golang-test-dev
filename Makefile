.PHONY: build run-api run-ingestion run-processor run-simulator up down clean test proto

# На Windows make обычно не установлен — используйте .\build.ps1 вместо make build

# Detect OS and set executable extension
ifeq ($(OS),Windows_NT)
    EXE_EXT := .exe
else
    EXE_EXT :=
endif

build:
	go mod download
	go build -o bin/api-gateway$(EXE_EXT) ./services/api-gateway
	go build -o bin/data-ingestion$(EXE_EXT) ./services/data-ingestion
	go build -o bin/alert-processor$(EXE_EXT) ./services/alert-processor
	go build -o bin/simulator$(EXE_EXT) ./simulator

run-api:
	./bin/api-gateway$(EXE_EXT)

run-ingestion:
	./bin/data-ingestion$(EXE_EXT)

run-processor:
	./bin/alert-processor$(EXE_EXT)

run-simulator:
	./bin/simulator$(EXE_EXT)

up:
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5

down:
	docker-compose down

clean:
	rm -rf bin/
	docker-compose down -v

proto:
	docker run --rm -v $$(pwd):/workspace -w /workspace bufbuild/buf:latest generate

test:
	@echo "Testing API Gateway..."
	@curl -s http://localhost:8080/health | grep -q "ok" && echo "✓ API Gateway is healthy" || echo "✗ API Gateway is not responding"
