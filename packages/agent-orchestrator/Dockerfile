# Build stage
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-orchestrator ./cmd/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

RUN addgroup -g 1000 -S orchestrator && \
    adduser -u 1000 -S orchestrator -G orchestrator

WORKDIR /app

COPY --from=builder /app/agent-orchestrator .

RUN chown -R orchestrator:orchestrator /app

USER orchestrator

EXPOSE 8083

ENV PORT=8083 \
    MAX_AGENTS=10 \
    MAX_TASKS_PER_AGENT=5 \
    LOG_LEVEL=info

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8083/health || exit 1

ENTRYPOINT ["./agent-orchestrator"]