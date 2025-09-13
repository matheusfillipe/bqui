.PHONY: help build run test test-coverage test-emulator clean lint fmt vet deps install

# Default target
help: ## Show this help message
	@echo "bqui - BigQuery Terminal UI"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Build configuration
BINARY_NAME=bqui
BUILD_DIR=./build
CMD_DIR=./cmd/bqui
VERSION=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.1.0")
BUILD_TIME=$(shell date +%Y-%m-%dT%H:%M:%S%z)
LDFLAGS=-ldflags "-s -w"

# Build the binary
build: deps ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: deps ## Build binaries for all platforms
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	
	@echo "All builds completed!"

# Run the application
run: build ## Build and run the application
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Run directly without building binary
dev: ## Run the application in development mode
	@echo "Running $(BINARY_NAME) in dev mode..."
	@go run $(CMD_DIR)

# Run with specific project
run-project: build ## Run with a specific project (usage: make run-project PROJECT=my-project)
	@echo "Running $(BINARY_NAME) with project $(PROJECT)..."
	@$(BUILD_DIR)/$(BINARY_NAME) -project $(PROJECT)

# Testing
test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

test-short: ## Run tests excluding long-running ones
	@echo "Running short tests..."
	@go test -v -short ./pkg/...
	@go test -v -short ./internal/...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-emulator: ## Run tests with BigQuery emulator (takes longer)
	@echo "Running tests with BigQuery emulator..."
	@go test -v -timeout=10m ./internal/bigquery/...

benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem ./...

# Code quality
lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

# Dependencies
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@go mod verify

tidy: ## Tidy up go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

# Installation
install: build ## Install the binary to GOPATH/bin
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	@go install $(CMD_DIR)

install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean -testcache
	@go clean -modcache

# Docker (optional)
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .

docker-run: ## Run in Docker container
	@echo "Running in Docker..."
	@docker run --rm -it \
		-v ~/.config/gcloud:/root/.config/gcloud:ro \
		-v ~/.config/gcloud:/home/nonroot/.config/gcloud:ro \
		$(BINARY_NAME):$(VERSION)

# Release preparation
release-check: test lint vet ## Run all checks before release
	@echo "All checks passed! Ready for release."

# Development helpers
watch: ## Watch for changes and rebuild (requires entr)
	@echo "Watching for changes... (requires 'entr' to be installed)"
	@find . -name '*.go' | entr -r make dev

# Show project info
info: ## Show project information
	@echo "Project: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Build dir: $(BUILD_DIR)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

# Check if required tools are installed
check-tools: ## Check if required development tools are installed
	@echo "Checking required tools..."
	@which go >/dev/null || (echo "Go is not installed" && exit 1)
	@which git >/dev/null || (echo "Git is not installed" && exit 1)
	@which gcloud >/dev/null || echo "Warning: gcloud CLI not found (needed for authentication)"
	@which golangci-lint >/dev/null || echo "Warning: golangci-lint not found (run 'make install-tools')"
	@which goimports >/dev/null || echo "Warning: goimports not found (run 'make install-tools')"
	@echo "Tool check completed!"

# Database/emulator helpers
start-emulator: ## Start BigQuery emulator for testing
	@echo "Starting BigQuery emulator..."
	@docker run --rm -p 9050:9050 \
		ghcr.io/goccy/bigquery-emulator:latest \
		--project=test-project \
		--dataset=test_dataset

stop-emulator: ## Stop BigQuery emulator
	@echo "Stopping BigQuery emulator..."
	@docker stop $$(docker ps -q --filter ancestor=ghcr.io/goccy/bigquery-emulator:latest) 2>/dev/null || echo "No emulator running"