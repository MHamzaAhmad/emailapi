# Base stage for dependencies
FROM golang:1.25-alpine AS base

WORKDIR /app

# Install system dependencies
RUN apk add --no-cache git make

# Copy go mod and sum files
COPY apps/api/go.mod apps/api/go.sum ./apps/api/

# Download dependencies
WORKDIR /app/apps/api
RUN go mod download

# Return to root
WORKDIR /app

# Development stage
FROM base AS dev

# Install air for live reloading
RUN go install github.com/air-verse/air@latest

# Copy source code
COPY . .

EXPOSE 8080 9090

CMD ["air", "-c", "apps/api/.air.toml"]

# Builder stage for production
FROM base AS builder

# Copy source code
COPY . .

# Build the application
WORKDIR /app/apps/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server main.go

# Production runner stage
FROM alpine:latest AS prod

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/bin/server /app/server

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["/app/server"]
