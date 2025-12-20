# Builder stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod and sum files
COPY apps/api/go.mod apps/api/go.sum ./apps/api/

# Download dependencies
WORKDIR /app/apps/api
RUN go mod download

# Return to root and copy source code
WORKDIR /app
COPY . .

# Build the application
WORKDIR /app/apps/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server main.go

# Production runner stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy binary from builder
COPY --from=builder /app/bin/server /app/server

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080 9090

CMD ["/app/server"]
