.DEFAULT_GOAL := help
REDIS_URL ?= redis://localhost:6379/0
EMULATOR  ?= 127.0.0.1:9099
WHEELCTL  ?= go run ./cmd/wheelctl

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: dev
dev: ## Run everything: Docker deps (detached) + API + web, seeded with demo wheels
	./scripts/dev.sh

.PHONY: setup
setup: ## Install toolchain (mise) and frontend dependencies
	mise install
	cd web && npm ci

.PHONY: gen
gen: ## Generate Go and TypeScript code from proto/
	buf lint
	buf generate

.PHONY: deps
deps: ## Start Redis and the Firebase Auth emulator in Docker
	docker compose up -d --build redis firebase

.PHONY: down
down: ## Stop Docker services (wheels in Redis are discarded)
	docker compose --profile server down

.PHONY: server
server: ## Run the API on :7755 against the auth emulator
	REDIS_URL=$(REDIS_URL) FIREBASE_AUTH_EMULATOR_HOST=$(EMULATOR) go run ./cmd/server

.PHONY: web
web: ## Run the Vite dev server on :5173
	cd web && npm run dev

.PHONY: seed
seed: ## Create a few demo wheels with wheelctl
	FIREBASE_AUTH_EMULATOR_HOST=$(EMULATOR) $(WHEELCTL) create -name "Lunch picker" -mode equal \
		-s Pizza -s Sushi -s Burritos -s Ramen -s Salad -s "Chef's choice"
	FIREBASE_AUTH_EMULATOR_HOST=$(EMULATOR) $(WHEELCTL) create -name "Loot box" -mode weighted \
		-s Common:60 -s Rare:30 -s Epic:9 -s Legendary:1
	FIREBASE_AUTH_EMULATOR_HOST=$(EMULATOR) $(WHEELCTL) create -name "Office raffle" -mode inventory \
		-s "Gift card::3" -s "Headphones::1" -s "Coffee voucher::10" -s "Sticker pack::25"

.PHONY: test
test: ## Run Go and frontend tests
	go test -race ./...
	cd web && npm test

.PHONY: lint
lint: ## Lint proto, Go and typecheck the frontend
	buf lint
	go vet ./...
	golangci-lint run
	cd web && npm run typecheck

.PHONY: build
build: ## Build Go binaries into bin/ and the frontend into web/dist
	go build -o bin/ ./cmd/...
	cd web && npm run build
