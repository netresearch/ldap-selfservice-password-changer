.PHONY: help
help: ## Show available targets
	@awk 'BEGIN{FS=":.*##";print "\n╔════════════════════════════════════════════════════════════════╗"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "║  \033[36m%-22s\033[0m %s\n", $$1, $$2} END{print "╚════════════════════════════════════════════════════════════════╝\n"}' $(MAKEFILE_LIST)

.PHONY: install
install: ## Install all dependencies (Node + Go)
	@echo "📦 Installing dependencies..."
	pnpm install
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
	pnpm dev

.PHONY: build
build: ## Build production binary
	@echo "🔨 Building production binary..."
	pnpm build
	@echo "✅ Build complete: ./ldap-selfservice-password-changer"

.PHONY: build-docker
build-docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t ldap-selfservice-password-changer:latest .
	@echo "✅ Docker image built"

.PHONY: test
test: ## Run all tests
	@echo "🧪 Running Go tests..."
	go test -v ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	@echo "📊 Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

.PHONY: typecheck
typecheck: ## Type check TypeScript
	@echo "🔍 Type checking TypeScript..."
	pnpm js:build
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
	pnpm prettier --write .
	go fmt ./...
	@echo "✅ Code formatted"

.PHONY: format-check
format-check: ## Check code formatting (CI)
	@echo "🔍 Checking code formatting..."
	pnpm prettier --check .
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "❌ Go files need formatting" && exit 1)
	@echo "✅ Code formatting check passed"

.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	rm -f ldap-selfservice-password-changer
	rm -f coverage.out coverage.html
	rm -rf node_modules/.cache tmp/
	@echo "✅ Cleaned"

.PHONY: docker-up
docker-up: ## Start Docker Compose services (dev profile)
	@echo "🐳 Starting Docker Compose services..."
	docker compose --profile dev up

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
