BINARY  := tweed
PKG     := ./cmd/$(BINARY)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build ./bin/tweed
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) $(PKG)

.PHONY: install
install: build ## Install to ~/.local/bin
	install -Dm755 bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)

.PHONY: test
test: ## Run unit tests with the race detector
	go test -race -cover ./...

.PHONY: cover
cover: ## Open an HTML coverage report
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format code
	gofmt -s -w .

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: check
check: tidy fmt lint test ## Everything CI runs; use before pushing

.PHONY: dry-run
dry-run: build ## Preview the workstation profile without changing anything
	./bin/$(BINARY) apply --profile workstation --dry-run

.PHONY: fedora-test
fedora-test: ## Dry-run inside a fresh Fedora container (needs podman)
	podman run --rm -v $(CURDIR):/src:Z -w /src registry.fedoraproject.org/fedora:latest \
		sh -c 'dnf install -y golang >/dev/null && go run $(PKG) apply -p workstation --dry-run'

.PHONY: snapshot
snapshot: ## Build release archives locally (needs goreleaser)
	goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Remove build output
	rm -rf bin dist coverage.out coverage.html
