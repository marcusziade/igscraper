# Makefile for igscraper

# Variables
BINARY_NAME := igscraper
DOCKER_IMAGE := igscraper
DOCKER_REGISTRY := ghcr.io
DOCKER_USERNAME := $(shell echo ${GITHUB_REPOSITORY} | cut -d'/' -f1)
GO_VERSION := 1.23
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -ldflags "-w -s -X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.commitHash=${COMMIT_HASH}"
BUILD_FLAGS := -v -trimpath

# Directories
DIST_DIR := dist
COVERAGE_DIR := coverage

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Platform detection
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Linux)
	OS := linux
endif
ifeq ($(UNAME_S),Darwin)
	OS := darwin
endif
ifeq ($(UNAME_M),x86_64)
	ARCH := amd64
endif
ifeq ($(UNAME_M),arm64)
	ARCH := arm64
endif

# Default target
.DEFAULT_GOAL := help

# Help target
.PHONY: help
help: ## Display this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\033[36m%-20s\033[0m %s\n", "Target", "Description"} /^[a-zA-Z_-]+:.*?##/ { printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Development targets
.PHONY: dev
dev: ## Run the application in development mode
	$(GOCMD) run ./cmd/igscraper

.PHONY: build
build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME) for $(OS)/$(ARCH)..."
	@mkdir -p $(DIST_DIR)
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/igscraper

.PHONY: build-all
build-all: ## Build binaries for all platforms
	@echo "Building binaries for all platforms..."
	@mkdir -p $(DIST_DIR)
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			echo "Building for $$os/$$arch..."; \
			GOOS=$$os GOARCH=$$arch $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) \
				-o $(DIST_DIR)/$(BINARY_NAME)-$$os-$$arch$$( [ $$os = "windows" ] && echo ".exe" ) \
				./cmd/igscraper; \
		done; \
	done

.PHONY: install
install: ## Install the binary to $GOPATH/bin
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) ./cmd/igscraper

.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR) $(COVERAGE_DIR)
	@go clean -cache -testcache

# Testing targets
.PHONY: test
test: ## Run unit tests
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	$(GOTEST) -v -race -tags=integration ./...

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Code quality targets
.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	$(GOFMT) -s -w .

.PHONY: lint
lint: ## Run linter
	@echo "Running linter..."
	@if ! which $(GOLINT) > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	$(GOLINT) run ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GOCMD) vet ./...

.PHONY: security
security: ## Run security checks
	@echo "Running security checks..."
	@if ! which gosec > /dev/null; then \
		echo "Installing gosec..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
	fi
	gosec -fmt=json -out=security-report.json ./... || true

.PHONY: check
check: fmt lint vet test ## Run all checks (fmt, lint, vet, test)

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	$(GOMOD) tidy
	$(GOGET) -u ./...
	$(GOMOD) tidy

.PHONY: deps-verify
deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

.PHONY: docker-build-multi
docker-build-multi: ## Build multi-platform Docker image
	@echo "Building multi-platform Docker image..."
	docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 \
		-t $(DOCKER_IMAGE):$(VERSION) -t $(DOCKER_IMAGE):latest .

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "Pushing Docker image to registry..."
	docker tag $(DOCKER_IMAGE):$(VERSION) $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(DOCKER_IMAGE):$(VERSION)
	docker tag $(DOCKER_IMAGE):latest $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(DOCKER_IMAGE):latest
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(DOCKER_IMAGE):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(DOCKER_IMAGE):latest

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run --rm -it \
		-v $(PWD)/downloads:/downloads \
		-v $(PWD)/config:/config \
		$(DOCKER_IMAGE):latest

.PHONY: docker-compose-up
docker-compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	docker-compose up -d

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	docker-compose down

.PHONY: docker-compose-logs
docker-compose-logs: ## Show docker-compose logs
	docker-compose logs -f

# Release targets
.PHONY: release-dry-run
release-dry-run: ## Perform a dry run of goreleaser
	@echo "Running goreleaser dry run..."
	@if ! which goreleaser > /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	goreleaser release --snapshot --skip-publish --clean

.PHONY: release
release: ## Create a new release
	@echo "Creating release..."
	@if ! which goreleaser > /dev/null; then \
		echo "Installing goreleaser..."; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	goreleaser release --clean

.PHONY: tag
tag: ## Create a new tag
	@echo "Current version: $(VERSION)"
	@read -p "Enter new version (e.g., v1.0.0): " NEW_VERSION; \
	git tag -a $$NEW_VERSION -m "Release $$NEW_VERSION"; \
	echo "Created tag $$NEW_VERSION"

# Documentation targets
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	@if ! which godoc > /dev/null; then \
		echo "Installing godoc..."; \
		go install golang.org/x/tools/cmd/godoc@latest; \
	fi
	godoc -http=:6060

.PHONY: docs-preview
docs-preview: ## Preview landing page
	@./scripts/preview-docs.sh

.PHONY: changelog
changelog: ## Generate changelog
	@echo "Generating changelog..."
	@if ! which git-cliff > /dev/null; then \
		echo "Please install git-cliff: https://github.com/orhun/git-cliff"; \
		exit 1; \
	fi
	git-cliff --output CHANGELOG.md

# Utility targets
.PHONY: setup
setup: ## Setup development environment
	@echo "Setting up development environment..."
	@echo "Installing tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/goreleaser/goreleaser@latest
	go install golang.org/x/tools/cmd/godoc@latest
	@echo "Installing pre-commit hooks..."
	@if which pre-commit > /dev/null; then \
		pre-commit install; \
	else \
		echo "pre-commit not found. Install it with: pip install pre-commit"; \
	fi
	@echo "Setup complete!"

.PHONY: info
info: ## Display project information
	@echo "Project: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(GO_VERSION)"
	@echo "Platform: $(OS)/$(ARCH)"

# CI/CD targets
.PHONY: ci-lint
ci-lint: ## Run CI linting
	$(GOLINT) run --timeout=5m ./...

.PHONY: ci-test
ci-test: ## Run CI tests
	$(GOTEST) -v -race -coverprofile=coverage.txt -covermode=atomic ./...

.PHONY: ci-build
ci-build: ## Run CI build
	CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/igscraper