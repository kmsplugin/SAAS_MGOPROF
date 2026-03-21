FROM golang:1.23-alpine AS builder

WORKDIR /app

# Зависимости
COPY go.mod go.sum ./
RUN go mod download

# Исходники
COPY . .

# Сборка
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ai-service ./cmd/server

# ── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /ai-service .

EXPOSE 8020

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:8020/health || exit 1

ENTRYPOINT ["/app/ai-service"]
