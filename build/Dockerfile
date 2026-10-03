# Этап сборки
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/bnn ./cmd/main

# Этап запуска
FROM scratch AS runner

WORKDIR /app/

COPY --from=builder /out/bnn ./bnn

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip /
ENV TZ="Europe/Moscow"
ENV ZONEINFO=/zoneinfo.zip

USER 65534:65534

EXPOSE 5458

ENTRYPOINT ["./bnn"]