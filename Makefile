APP_NAME = nexora
GO_FILES = $(shell find . -name '*.go' -not -path './vendor/*' -not -path './site/*' -not -path './web/*')
GO_PKGS = $(shell go list ./cmd/... ./internal/...)

.PHONY: help build run dev test lint clean migrate-up migrate-down migrate-create docker-up docker-down

help:
	@echo "Nexora CMS - Comandos de desenvolvimento"
	@echo ""
	@echo "Uso:"
	@echo "  make build          Compilar o binário da API"
	@echo "  make run            Executar a API localmente"
	@echo "  make dev            Iniciar ambiente de desenvolvimento (Docker)"
	@echo "  make test           Executar testes"
	@echo "  make lint           Executar linter"
	@echo "  make clean          Limpar artefatos de build"
	@echo "  make migrate-up     Aplicar migrations"
	@echo "  make migrate-down   Reverter última migration"
	@echo "  make migrate-create Criar nova migration"
	@echo "  make docker-up      Iniciar containers Docker"
	@echo "  make docker-down    Parar containers Docker"

build:
	go build -o bin/$(APP_NAME) ./cmd/api

run:
	go run ./cmd/api/main.go

dev:
	docker compose -f deploy/docker-compose.yml up --build

test:
	go test ./... -v -count=1

test-race:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

clean:
	rm -rf bin/ dist/ tmp/
	go clean -cache

migrate-up:
	go run ./cmd/migrate/main.go up

migrate-down:
	go run ./cmd/migrate/main.go down $(steps)

migrate-create:
	@read -p "Migration name: " name; \
	go run ./cmd/migrate/main.go create $$name

docker-up:
	docker compose -f deploy/docker-compose.yml up -d

docker-down:
	docker compose -f deploy/docker-compose.yml down

docker-logs:
	docker compose -f deploy/docker-compose.yml logs -f

sqlc-gen:
	sqlc generate

tidy:
	go mod tidy

.PHONY: help build run dev test test-race lint lint-fix clean
.PHONY: migrate-up migrate-down migrate-create
.PHONY: docker-up docker-down docker-logs sqlc-gen tidy
