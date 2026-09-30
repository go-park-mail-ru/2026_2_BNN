# Этап сборки
FROM golang:1.26-alpine AS builder

WORKDIR /github.com/go-park-mail-ru/2026_2_BNN/

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ./.bin ./cmd/main/main.go

# Этап запуска
FROM scratch AS runner

WORKDIR /app/

COPY --from=builder /github.com/go-park-mail-ru/2026_2_BNN/.bin .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /
ENV TZ="Europe/Moscow"
ENV ZONEINFO=/zoneinfo.zip

EXPOSE 5458

ENTRYPOINT ["./.bin"]