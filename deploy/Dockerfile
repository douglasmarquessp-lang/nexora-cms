# ===== Stage 1: Build =====
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

# Download dependencies first for better Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy project
COPY . .

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


# ===== Stage 2: Development =====
FROM golang:1.26-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]


# ===== Stage 3: Production =====
FROM alpine:3.20 AS prod

RUN apk add --no-cache \
    ca-certificates \
    tzdata

WORKDIR /app

# API binary
COPY --from=builder /build/nexora ./nexora

# Migration binary
COPY --from=builder /build/migrate ./migrate

# Database migrations
COPY migrations/ ./migrations/

# Ensure binaries are executable
RUN chmod +x ./nexora ./migrate

EXPOSE 8080

# Start API
ENTRYPOINT ["./nexora"]
