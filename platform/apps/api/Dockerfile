FROM golang:1.23-alpine AS builder
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY internal/stub/ internal/stub/

# Download dependencies (offline-friendly)
RUN go mod download || true

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/api ./cmd/server

# Final image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/api .

EXPOSE 8080
ENTRYPOINT ["/app/api"]
