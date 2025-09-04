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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o capsule-builder .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/capsule-builder .

# Expose port
EXPOSE 8092

# Run the application
CMD ["./capsule-builder"]