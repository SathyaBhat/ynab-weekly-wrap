.PHONY: help build run test clean lint fmt vet docker-build docker-run docker-compose-up docker-compose-down

DOCKER_USERNAME := sathyabhat
# Variables
APP_NAME := ynab-weekly-wrap
DOCKER_IMAGE := $(APP_NAME):latest
DOCKER_REGISTRY := $(DOCKER_USERNAME)/$(APP_NAME)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the application
	@echo "Building $(APP_NAME)..."
	go build $(LDFLAGS) -o bin/$(APP_NAME) ./cmd/app

build-linux: ## Build for Linux
	@echo "Building $(APP_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-linux ./cmd/app

build-macos: ## Build for macOS
	@echo "Building $(APP_NAME) for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(APP_NAME)-macos ./cmd/app

build-all: build-linux build-macos ## Build for all platforms

run: build ## Build and run the application
	@echo "Running $(APP_NAME)..."
	./bin/$(APP_NAME)

test: ## Run tests
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "Coverage report:"
	@go tool cover -func=coverage.out | tail -1

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run golangci-lint
	@echo "Running linter..."
	golangci-lint run ./...

fmt: ## Format code
	@echo "Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	go mod tidy

docker-build: ## Build Docker image
	@echo "Building Docker image: $(DOCKER_IMAGE)"
	docker build -t $(DOCKER_IMAGE) -t $(DOCKER_REGISTRY):$(VERSION) .

docker-run: docker-build ## Build and run Docker container
	@echo "Running Docker container..."
	docker run --rm \
		--name $(APP_NAME) \
		--env-file .env \
		-v $(PWD)/logs:/app/logs \
		$(DOCKER_IMAGE)

docker-compose-up: ## Start services with docker-compose
	@echo "Starting docker-compose services..."
	docker-compose up -d
	@echo "Services started. Check logs with: docker-compose logs -f"

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping docker-compose services..."
	docker-compose down

docker-compose-logs: ## View docker-compose logs
	docker-compose logs -f

docker-push: docker-build ## Build and push Docker image to registry
	@echo "Pushing Docker image to registry..."
	docker push $(DOCKER_REGISTRY):$(VERSION)
	docker push $(DOCKER_REGISTRY):latest

install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

version: ## Display version information
	@echo "Application: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

dev: ## Run in development mode with hot reload (requires air)
	@echo "Running in development mode..."
	air

all: clean deps build test ## Clean, download deps, build, and test

.DEFAULT_GOAL := help
