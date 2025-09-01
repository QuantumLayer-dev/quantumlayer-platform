# Build stage
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY packages/llm-router/go.mod packages/llm-router/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY packages/llm-router/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o llm-router ./cmd/main.go

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S llmrouter && \
    adduser -u 1000 -S llmrouter -G llmrouter

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/llm-router .

# Change ownership
RUN chown -R llmrouter:llmrouter /app

# Switch to non-root user
USER llmrouter

# Expose ports
EXPOSE 8080 9090

# Set environment defaults
ENV PORT=8080 \
    METRICS_PORT=9090 \
    LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
ENTRYPOINT ["./llm-router"]