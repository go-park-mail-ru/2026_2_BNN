# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/bin/bnn \
    ./cmd/main


FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/bin/bnn /app/bnn
COPY --from=builder /app/static /app/static
USER appuser
EXPOSE 5458
CMD ["/app/bnn"]