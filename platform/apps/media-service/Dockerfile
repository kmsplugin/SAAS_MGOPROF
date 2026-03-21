FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
COPY internal/stub/ internal/stub/
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/media-service ./cmd/server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /bin/media-service .

EXPOSE 8010
ENTRYPOINT ["/app/media-service"]
