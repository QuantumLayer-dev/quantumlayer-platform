# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod download && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o image-registry .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/image-registry .

EXPOSE 8096

CMD ["./image-registry"]