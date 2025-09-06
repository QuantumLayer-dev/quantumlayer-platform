# Production Dockerfile for Meta Prompt Engine
# Enterprise-grade, multi-stage build

FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy all source files
COPY . ./

# Download dependencies
RUN go mod download

# Build arguments
ARG VERSION=latest
ARG BUILD_TIME

# Build the binary with proper flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o server \
    ./cmd/server

# Final production image
FROM gcr.io/distroless/static:nonroot

# Copy timezone and certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /build/server /server

# Use non-root user
USER nonroot:nonroot

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "--health-check"]

# Metadata
LABEL org.opencontainers.image.source="https://github.com/QuantumLayer-dev/quantumlayer-platform" \
      org.opencontainers.image.title="Meta Prompt Engine" \
      org.opencontainers.image.description="Enterprise AI Prompt Optimization Engine" \
      org.opencontainers.image.vendor="QuantumLayer"

ENTRYPOINT ["/server"]