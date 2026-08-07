# ===== Nexora CMS — build de produção (Railway / Docker) =====
# Um único serviço: a API Go serve o Admin SPA compilado (Vite) embutido
# no binário via go:embed (ver internal/webui). O Dockerfile é o canônico;
# o deploy/docker-compose.yml reutiliza os stages dev para desenvolvimento.

# ===== Stage 1: Build Admin SPA (Vite) =====
FROM node:22-alpine AS web-builder

WORKDIR /build/web

# Dependências primeiro (layer cache)
COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# ===== Stage 2: Build binários Go =====
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Injeta o build do Admin SPA dentro do diretório embutido da API
# (mantém o .gitkeep do repo, adiciona os arquivos reais do build).
COPY --from=web-builder /build/web/dist ./internal/webui/dist/

# Build API binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/nexora \
    ./cmd/api/main.go

# Build migration binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" \
    -o /build/migrate \
    ./cmd/migrate/main.go

# ===== Stage 3: Development (docker-compose) =====
FROM golang:1.26-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]

# ===== Stage 4: Production =====
FROM alpine:3.20 AS prod

RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /app

# API binary (contém o Admin SPA embutido)
COPY --from=builder /build/nexora ./nexora

# Migration binary
COPY --from=builder /build/migrate ./migrate

# Database migrations
COPY migrations/ ./migrations/

# Ensure binaries are executable
RUN chmod +x ./nexora ./migrate

EXPOSE 8080

# Start API (migrations rodam no startup; o binário serve API + /admin)
ENTRYPOINT ["./nexora"]