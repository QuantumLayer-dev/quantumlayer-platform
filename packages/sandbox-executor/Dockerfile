# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sandbox-executor .

# Runtime stage
FROM docker:24-dind

# Install necessary packages
RUN apk add --no-cache \
    ca-certificates \
    bash \
    curl

# Copy the binary from builder
COPY --from=builder /app/sandbox-executor /usr/local/bin/

# Create workspace directory
RUN mkdir -p /workspace

WORKDIR /workspace

# Expose port
EXPOSE 8091

# Start Docker daemon and the application
ENTRYPOINT ["sh", "-c", "dockerd-entrypoint.sh & sleep 5 && sandbox-executor"]