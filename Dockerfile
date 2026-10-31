# Construction stage
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o main ./cmd/main.go

# Final stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app -h /home/app app \
    && mkdir -p /home/app \
    && chown app:app /home/app

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
COPY database/migrations/ /app/database/migrations/

# Runtime configuration comes from the environment (compose env_file or docker run -e), never from the image.
ENV MIGRATIONS_PATH=file:///app/database/migrations

USER app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD wget -qO- "http://127.0.0.1:${APP_PORT:-8080}/ping" >/dev/null || exit 1

CMD ["./main"]
