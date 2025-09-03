# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy all required files
COPY packages/parser/ ./
COPY packages/shared/ ./shared/

# Fix module replacement if needed
RUN if [ -f go.mod ]; then go mod edit -replace github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared=./shared || true; fi

# Download dependencies and tidy
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o parser ./cmd/server/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/parser .

EXPOSE 8082

ENTRYPOINT ["./parser"]
