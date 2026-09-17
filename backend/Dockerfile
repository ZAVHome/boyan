# ==============================================================================
# Multi-stage Dockerfile для Go бэкенда Boyan
# ==============================================================================

# 1. Этап сборки
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src/backend

# Кэширование Go-модулей
COPY go.mod go.sum ./
RUN go mod download

# Копирование исходного кода
COPY . .

# Статическая компиляция бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /bin/boyan ./cmd/server

# 2. Финальный минимальный образ
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata curl bash

# Создание непривилегированного пользователя boyan
RUN addgroup -S boyan && adduser -S boyan -G boyan

WORKDIR /app

# Создание необходимых рабочих директорий
RUN mkdir -p /app/data /books /import /app/backups /app/cache/covers && \
    chown -R boyan:boyan /app /books /import

# Копирование скомпилированного бинарника
COPY --from=builder /bin/boyan /app/boyan

USER boyan

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=5s --retries=3 --start-period=5s \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/boyan"]
CMD ["--config", "/app/config.yaml"]
