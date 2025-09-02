# Production Dockerfile for API Gateway
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy and download dependencies
COPY packages/api-gateway/ ./
COPY packages/shared/ ./shared/

RUN go mod edit -replace github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared=./shared
RUN go mod download

# Build
ARG VERSION=latest
ARG BUILD_TIME

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o server \
    cmd/server/main.go

# Final image
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/server /server

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]