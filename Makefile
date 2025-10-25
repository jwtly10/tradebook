.PHONY: run test integration-test lint build

run:
	@echo "Running Tradebook..."
	@go run ./cmd/tradebook/main.go

test:
	@echo "Running unit tests..."
	@go test -v ./...

coverage:
	@echo "Generating test coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html. Opening"
	open coverage.html

integration-test:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

lint:
	@echo "Running linters..."
	@golangci-lint run

push: lint test integration-test
	@echo "Running pre push checks"

all: lint test integration-test build
	@echo "All tasks completed successfully."
