.PHONY: help
help: ## Show available targets
	@awk 'BEGIN{FS=":.*##";print "\n╔════════════════════════════════════════════════════════════════╗"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "║  \033[36m%-22s\033[0m %s\n", $$1, $$2} END{print "╚════════════════════════════════════════════════════════════════╝\n"}' $(MAKEFILE_LIST)

.PHONY: install
install: ## Install all dependencies (Node + Go)
	@echo "📦 Installing dependencies..."
	bun install
	go mod download
	@echo "✅ Dependencies installed"

.PHONY: hooks
hooks: ## Install git hooks
	@echo "🔗 Installing git hooks..."
	cp githooks/* .git/hooks/
	chmod +x .git/hooks/pre-commit .git/hooks/commit-msg
	@echo "✅ Git hooks installed"

.PHONY: setup
setup: install hooks ## Full setup: install deps and git hooks

.PHONY: dev
dev: ## Start development servers with hot-reload
	@echo "🚀 Starting development mode..."
	bun run dev

.PHONY: build
build: ## Build production binary
	@echo "🔨 Building production binary..."
	bun run build
	@echo "✅ Build complete: ./ldap-selfservice-password-changer"

.PHONY: build-docker
build-docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t ldap-selfservice-password-changer:latest .
	@echo "✅ Docker image built"

.PHONY: test
test: ## Run all unit tests
	@echo "🧪 Running Go tests..."
	go test -v ./...

.PHONY: test-unit
test-unit: ## Run unit tests with race detection
	@echo "🧪 Running unit tests..."
	go test -v -race ./...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker services)
	@echo "🧪 Running integration tests..."
	go test -v -race -tags=integration ./...

.PHONY: test-e2e
test-e2e: ## Run end-to-end tests
	@echo "🧪 Running E2E tests..."
	go test -v -race -tags=e2e ./e2e/...

.PHONY: test-fuzz
test-fuzz: ## Run fuzz tests (30s per target)
	@echo "🧪 Running fuzz tests..."
	go test -fuzz=FuzzValidateNewPassword -fuzztime=30s ./internal/rpchandler/...
	go test -fuzz=FuzzPluralize -fuzztime=30s ./internal/rpchandler/...
	go test -fuzz=FuzzValidateEmailAddress -fuzztime=30s ./internal/email/...
	go test -fuzz=FuzzExtractClientIP -fuzztime=30s ./internal/rpchandler/...
	go test -fuzz=FuzzTokenStore -fuzztime=30s ./internal/resettoken/...

.PHONY: test-fuzz-quick
test-fuzz-quick: ## Run quick fuzz tests (5s per target)
	@echo "🧪 Running quick fuzz tests..."
	go test -fuzz=FuzzValidateNewPassword -fuzztime=5s ./internal/rpchandler/...
	go test -fuzz=FuzzPluralize -fuzztime=5s ./internal/rpchandler/...
	go test -fuzz=FuzzValidateEmailAddress -fuzztime=5s ./internal/email/...

.PHONY: test-mutation
test-mutation: ## Run mutation tests with gremlins (optional)
	@echo "🧪 Running mutation tests..."
	@command -v gremlins >/dev/null 2>&1 || { echo "gremlins not installed. Install with: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest"; exit 1; }
	gremlins unleash ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

.PHONY: test-all
test-all: test-unit test-fuzz-quick ## Run unit and quick fuzz tests
	@echo "✅ All tests passed"

.PHONY: typecheck
typecheck: ## Type check TypeScript
	@echo "🔍 Type checking TypeScript..."
	bun run js:build
	@echo "✅ TypeScript type check passed"

.PHONY: lint
lint: ## Run linters
	@echo "🔍 Running linters..."
	@echo "  → Go vet..."
	go vet ./...
	@echo "✅ Linting complete"

.PHONY: format
format: ## Format all code
	@echo "✨ Formatting code..."
	bunx prettier --write .
	go fmt ./...
	@echo "✅ Code formatted"

.PHONY: format-check
format-check: ## Check code formatting (CI)
	@echo "🔍 Checking code formatting..."
	bunx prettier --check .
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "❌ Go files need formatting" && exit 1)
	@echo "✅ Code formatting check passed"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -f ldap-selfservice-password-changer
	rm -f coverage.out coverage.html
	rm -rf node_modules/.cache tmp/
	@echo "✅ Cleaned"

# The Dockerfile only selects pre-built binaries out of bin/ (see the development
# guide, "The Dockerfile Is a Binary-Selector"), so the Compose stack needs the
# Linux binary for the host architecture built first. Assets go before the Go
# build because they are embedded via go:embed.
GOARCH ?= $(shell go env GOARCH)

.PHONY: build-linux
build-linux: ## Build the Linux binary the Dockerfile copies from bin/
	@echo "🔨 Building bin/ldap-selfservice-password-changer-linux-$(GOARCH)..."
	bun install --frozen-lockfile
	bun run build:assets
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GOARCH) go build -trimpath \
		-ldflags="-w -s -X main.version=dev -X main.build=$$(git rev-parse --short HEAD)" \
		-o bin/ldap-selfservice-password-changer-linux-$(GOARCH) .

.PHONY: docker-up
docker-up: build-linux ## Build the binary, then start Docker Compose services (dev profile)
	@echo "🐳 Starting Docker Compose services..."
	docker compose --profile dev up --build

.PHONY: docker-down
docker-down: ## Stop Docker Compose services
	@echo "🛑 Stopping Docker Compose services..."
	docker compose down

.PHONY: docker-logs
docker-logs: ## Show Docker Compose logs
	docker compose logs -f

.PHONY: docs
docs: ## Open documentation index
	@echo "📖 Documentation available at: ./docs/README.md"
	@echo ""
	@echo "  📚 Available guides:"
	@echo "    - docs/development-guide.md   (setup & workflows)"
	@echo "    - docs/api-reference.md        (JSON-RPC API)"
	@echo "    - docs/testing-guide.md        (testing strategies)"
	@echo "    - docs/accessibility.md        (WCAG 2.2 AAA)"
	@echo "    - docs/architecture.md         (system overview)"
	@echo ""
	@echo "  🤖 Agent guidelines: AGENTS.md, internal/AGENTS.md, internal/web/AGENTS.md"

.PHONY: ci
ci: format-check typecheck lint test ## Run all CI checks locally
	@echo "✅ All CI checks passed"
