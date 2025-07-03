# Enhanced Agent Makefile
# Implements AGENT_USE.md specification

.PHONY: help build run test clean deps validate version

# Variables
BINARY_NAME=enhanced-agent
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default config file
CONFIG_FILE=../configs/enhanced-agent.yaml

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

deps: ## Download Go module dependencies
	go mod download
	go mod tidy

build: deps ## Build the enhanced agent binary
	go build $(LDFLAGS) -o $(BINARY_NAME) .

run: build ## Run the enhanced agent with default config
	./$(BINARY_NAME) --config $(CONFIG_FILE) --debug

run-prod: build ## Run the enhanced agent in production mode
	./$(BINARY_NAME) --config $(CONFIG_FILE)

validate: build ## Validate the configuration
	./$(BINARY_NAME) validate --config $(CONFIG_FILE)

version: build ## Show version information
	./$(BINARY_NAME) version

test: ## Run tests
	go test -v ./...

test-race: ## Run tests with race detection
	go test -race -v ./...

clean: ## Clean build artifacts
	rm -f $(BINARY_NAME)
	rm -rf agent-data/logs/*
	rm -rf agent-data/cache/*

# Development targets
dev-setup: ## Set up development environment
	mkdir -p agent-data/{logs,plugins,cache,config/certificates}
	cp ../configs/enhanced-agent.yaml ./config.yaml
	@echo "Development environment set up!"

dev-run: dev-setup build ## Set up and run in development mode
	./$(BINARY_NAME) --config config.yaml --debug

# Docker targets
docker-build: ## Build Docker image
	docker build -t stavily/enhanced-agent:$(VERSION) .

docker-run: ## Run Docker container
	docker run --rm -it \
		-v $(PWD)/config.yaml:/app/config.yaml \
		-v $(PWD)/agent-data:/app/agent-data \
		stavily/enhanced-agent:$(VERSION)

# Lint and format
fmt: ## Format Go code
	go fmt ./...

lint: ## Run golangci-lint
	golangci-lint run

# Install development tools
install-tools: ## Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show configuration
show-config: build ## Show current configuration
	./$(BINARY_NAME) validate --config $(CONFIG_FILE) || true

# Health check
health: ## Check if the agent is healthy (requires running agent)
	curl -f http://localhost:8081/health || echo "Agent not running or unhealthy"

# Metrics
metrics: ## Show agent metrics (requires running agent)
	curl -s http://localhost:9091/metrics | head -20

.DEFAULT_GOAL := help 