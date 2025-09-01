# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o semaroute-server ./cmd/semaroute-server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S semaroute && \
    adduser -u 1001 -S semaroute -G semaroute

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/semaroute-server .

# Copy configuration file
COPY --from=builder /app/config.yaml .

# Create logs directory
RUN mkdir -p logs && chown -R semaroute:semaroute logs

# Switch to non-root user
USER semaroute

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
CMD ["./semaroute-server"]
