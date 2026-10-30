# Construction stage
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o main ./cmd/main.go

# Final stage
FROM alpine:3.21

RUN addgroup -S app && adduser -S -G app app

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
COPY database/migrations/ /app/database/migrations/

# Runtime configuration comes from the environment (compose env_file or docker run -e), never from the image.
ENV MIGRATIONS_PATH=file:///app/database/migrations

USER app

EXPOSE 8080

CMD ["./main"]
