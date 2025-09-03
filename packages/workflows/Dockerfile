# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY packages/workflows/go.mod packages/workflows/go.sum ./
RUN go mod download

# Copy source code
COPY packages/workflows/ ./

# Build the worker
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/worker .

EXPOSE 9090

ENTRYPOINT ["./worker"]