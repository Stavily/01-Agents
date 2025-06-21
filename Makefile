# Makefile for Stavily Agents
.PHONY: help build test lint clean docker-build docker-push deploy-local install-deps

# Variables
GO_VERSION := 1.21
BINARY_DIR := bin
DOCKER_REGISTRY := stavily
VERSION ?= latest

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')"
BUILD_FLAGS := -trimpath $(LDFLAGS)

# Default target
help: ## Show this help message
	@echo "Stavily Agents Build System"
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: build-sensor build-action ## Build all agents

build-sensor: ## Build sensor agent
	@echo "Building sensor agent..."
	@mkdir -p $(BINARY_DIR)
	cd sensor-agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/sensor-agent cmd/sensor-agent/main.go

build-action: ## Build action agent
	@echo "Building action agent..."
	@mkdir -p $(BINARY_DIR)
	cd action-agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/action-agent cmd/action-agent/main.go

build-cross: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p $(BINARY_DIR)
	# Linux AMD64
	cd sensor-agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/sensor-agent-linux-amd64 cmd/sensor-agent/main.go
	cd action-agent && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/action-agent-linux-amd64 cmd/action-agent/main.go
	# Linux ARM64
	cd sensor-agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/sensor-agent-linux-arm64 cmd/sensor-agent/main.go
	cd action-agent && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/action-agent-linux-arm64 cmd/action-agent/main.go
	# macOS AMD64
	cd sensor-agent && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/sensor-agent-darwin-amd64 cmd/sensor-agent/main.go
	cd action-agent && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/action-agent-darwin-amd64 cmd/action-agent/main.go
	# Windows AMD64
	cd sensor-agent && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/sensor-agent-windows-amd64.exe cmd/sensor-agent/main.go
	cd action-agent && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o ../$(BINARY_DIR)/action-agent-windows-amd64.exe cmd/action-agent/main.go

# Testing targets
test: test-shared test-sensor test-action ## Run all tests

test-shared: ## Run shared library tests
	@echo "Running shared library tests..."
	cd shared && go test -v -race -cover ./...

test-sensor: ## Run sensor agent tests
	@echo "Running sensor agent tests..."
	cd sensor-agent && go test -v -race -cover ./...

test-action: ## Run action agent tests
	@echo "Running action agent tests..."
	cd action-agent && go test -v -race -cover ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	cd shared && go test -v -tags=integration ./...
	cd sensor-agent && go test -v -tags=integration ./...
	cd action-agent && go test -v -tags=integration ./...

# Code quality targets
lint: ## Run linting on all code
	@echo "Running linters..."
	cd shared && golangci-lint run
	cd sensor-agent && golangci-lint run
	cd action-agent && golangci-lint run

fmt: ## Format all Go code
	@echo "Formatting code..."
	cd shared && go fmt ./...
	cd sensor-agent && go fmt ./...
	cd action-agent && go fmt ./...

vet: ## Run go vet on all code
	@echo "Running go vet..."
	cd shared && go vet ./...
	cd sensor-agent && go vet ./...
	cd action-agent && go vet ./...

# Dependency management
deps: ## Download and tidy dependencies
	@echo "Managing dependencies..."
	cd shared && go mod download && go mod tidy
	cd sensor-agent && go mod download && go mod tidy
	cd action-agent && go mod download && go mod tidy

install-deps: ## Install development dependencies
	@echo "Installing development dependencies..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install golang.org/x/tools/cmd/goimports@latest

# Docker targets
docker-build: docker-build-sensor docker-build-action ## Build all Docker images

docker-build-sensor: ## Build sensor agent Docker image
	@echo "Building sensor agent Docker image..."
	docker build -f sensor-agent/Dockerfile -t $(DOCKER_REGISTRY)/sensor-agent:$(VERSION) .

docker-build-action: ## Build action agent Docker image
	@echo "Building action agent Docker image..."
	docker build -f action-agent/Dockerfile -t $(DOCKER_REGISTRY)/action-agent:$(VERSION) .

docker-push: ## Push Docker images to registry
	@echo "Pushing Docker images..."
	docker push $(DOCKER_REGISTRY)/sensor-agent:$(VERSION)
	docker push $(DOCKER_REGISTRY)/action-agent:$(VERSION)

# Local development
dev-sensor: ## Run sensor agent in development mode
	@echo "Starting sensor agent in development mode..."
	cd sensor-agent && go run cmd/sensor-agent/main.go --config configs/dev.yaml --log-level debug

dev-action: ## Run action agent in development mode
	@echo "Starting action agent in development mode..."
	cd action-agent && go run cmd/action-agent/main.go --config configs/dev.yaml --log-level debug

dev-compose: ## Start both agents with docker-compose
	@echo "Starting agents with docker-compose..."
	docker-compose -f deployments/docker/docker-compose.dev.yml up --build

# Deployment targets
deploy-local: build ## Deploy agents locally for testing
	@echo "Deploying agents locally..."
	./scripts/deploy-local.sh

package: build-cross ## Package agents for distribution
	@echo "Packaging agents..."
	./scripts/package.sh

# Cleanup targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	cd shared && go clean -cache -testcache
	cd sensor-agent && go clean -cache -testcache
	cd action-agent && go clean -cache -testcache

clean-docker: ## Clean Docker images
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_REGISTRY)/sensor-agent:$(VERSION) || true
	docker rmi $(DOCKER_REGISTRY)/action-agent:$(VERSION) || true

# Documentation
docs: ## Generate documentation
	@echo "Generating documentation..."
	cd sensor-agent && swag init -g cmd/sensor-agent/main.go -o docs/
	cd action-agent && swag init -g cmd/action-agent/main.go -o docs/

# Security scanning
security: ## Run security scans
	@echo "Running security scans..."
	cd shared && gosec ./...
	cd sensor-agent && gosec ./...
	cd action-agent && gosec ./...

# Performance benchmarks
bench: ## Run performance benchmarks
	@echo "Running benchmarks..."
	cd shared && go test -bench=. -benchmem ./...
	cd sensor-agent && go test -bench=. -benchmem ./...
	cd action-agent && go test -bench=. -benchmem ./...

# Release targets
release: clean build-cross package ## Create a release build
	@echo "Creating release build..."

# CI/CD targets
ci: install-deps deps fmt vet lint test security ## Run CI pipeline
	@echo "CI pipeline completed successfully"

# Version information
version: ## Show version information
	@echo "Go version: $(shell go version)"
	@echo "Build version: $(VERSION)"
	@echo "Build time: $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')" 