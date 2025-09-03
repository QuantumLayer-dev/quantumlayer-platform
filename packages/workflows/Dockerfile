# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . ./

# Download dependencies and build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/worker .

EXPOSE 9090

ENTRYPOINT ["./worker"]