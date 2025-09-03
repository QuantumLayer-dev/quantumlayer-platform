# Build stage
FROM golang:1.21-alpine AS builder

# Install swag for documentation generation
RUN go install github.com/swaggo/swag/cmd/swag@latest

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs
RUN swag init

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api-docs .

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/api-docs .
COPY --from=builder /app/docs ./docs

# Expose port
EXPOSE 8090

# Run the binary
CMD ["./api-docs"]