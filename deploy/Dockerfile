# ===== Stage 1: Build =====
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/nexora \
    ./cmd/api/main.go

# Build migration CLI
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /build/migrate \
    ./cmd/migrate


# ===== Stage 2: Dev =====
FROM golang:1.26-alpine AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]


# ===== Stage 3: Production =====
FROM alpine:3.20 AS prod

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# API binary
COPY --from=builder /build/nexora ./nexora

# Migration binary
COPY --from=builder /build/migrate ./migrate

# Database migrations
COPY migrations/ ./migrations/

EXPOSE 8080

ENTRYPOINT ["./nexora"]
