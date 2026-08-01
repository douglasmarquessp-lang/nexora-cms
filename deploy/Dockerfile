# ===== Stage 1: Build API =====
FROM golang:1.26-alpine AS api-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/nexora \
    ./cmd/api/main.go


# ===== Stage 2: Build Migration CLI =====
FROM golang:1.26-alpine AS migrate-builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/migrate \
    ./cmd/migrate


# ===== Stage 3: Production =====
FROM alpine:3.20 AS production

RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /app

# API
COPY --from=api-builder /build/nexora ./nexora

# Migration CLI
COPY --from=migrate-builder /build/migrate ./migrate

# SQL migrations
COPY migrations/ ./migrations/

# Railway injects PORT automatically.
EXPOSE 8080

# Start API
ENTRYPOINT ["./nexora"]
