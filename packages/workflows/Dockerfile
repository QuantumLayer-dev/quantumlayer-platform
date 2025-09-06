# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

# Copy entire project for build context
COPY . .

# Move to workflows directory
WORKDIR /build/packages/workflows

# Download dependencies and build
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/packages/workflows/worker .

EXPOSE 9090

ENTRYPOINT ["./worker"]