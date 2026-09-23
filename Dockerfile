# syntax=docker/dockerfile:1

# ---------- Этап 1: сборка ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Сначала копируем только файлы зависимостей — это кэшируется
COPY go.mod go.sum ./
RUN go mod download

# Затем исходники
COPY . .

# Собираем статический бинарник для Linux
# CGO_ENABLED=0 — без C-зависимостей (нужно для scratch/alpine)
# -ldflags="-s -w" — убираем отладочную информацию, бинарник меньше
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/bnn \
    ./cmd/main

# ---------- Этап 2: рантайм ----------
FROM alpine:3.20

# ca-certificates нужны, если приложение ходит по HTTPS наружу
# tzdata — если используешь time.LoadLocation
RUN apk --no-cache add ca-certificates tzdata

# Непривилегированный пользователь — базовая безопасность
RUN adduser -D -g '' appuser

WORKDIR /app

# Копируем бинарник из этапа сборки
COPY --from=builder /app/bin/bnn /app/bnn

# Копируем статику, если она нужна в рантайме (у тебя есть папка static)
COPY --from=builder /app/static /app/static

# Переключаемся на непривилегированного пользователя
USER appuser

# Порт, который слушает приложение (у тебя :5458)
EXPOSE 5458

# Запуск
CMD ["/app/bnn"]