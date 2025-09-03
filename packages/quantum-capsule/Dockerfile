# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Create go.mod file
RUN echo 'module github.com/QuantumLayer-dev/quantumlayer-platform/packages/quantum-capsule' > go.mod && \
    echo '' >> go.mod && \
    echo 'go 1.21' >> go.mod && \
    echo '' >> go.mod && \
    echo 'require (' >> go.mod && \
    echo '    github.com/gin-gonic/gin v1.9.1' >> go.mod && \
    echo '    github.com/google/uuid v1.3.1' >> go.mod && \
    echo ')' >> go.mod

# Copy source files
COPY . ./

# Download dependencies and build
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -o quantum-capsule main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/quantum-capsule .

EXPOSE 8090

ENTRYPOINT ["./quantum-capsule"]