.PHONY: run test integration-test lint build

run:
	@echo "Running Tradebook..."
	@go run ./cmd/tradebook/main.go

test:
	@echo "Running unit tests..."
	@go test -v ./...

integration-test:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

lint:
	@echo "Running linters..."
	@golangci-lint run

build: lint test
	@echo "Building the project..."
	@go build -o bin/tradebook ./cmd/tradebook/main.go

all: lint test integration-test build
	@echo "All tasks completed successfully."
