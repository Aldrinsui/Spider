# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum* ./
RUN go mod download || true

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /spider-api ./cmd/api

# Final runtime stage
FROM alpine:3.19

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /spider-api /app/spider-api
COPY --from=builder /app/internal/db/migrations /app/migrations

EXPOSE 8080

CMD ["/app/spider-api"]
