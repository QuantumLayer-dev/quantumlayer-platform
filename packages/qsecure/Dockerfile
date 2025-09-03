FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o qsecure-engine ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/qsecure-engine .

# Copy security rules and patterns
COPY --from=builder /app/rules ./rules
COPY --from=builder /app/patterns ./patterns

EXPOSE 8096

CMD ["./qsecure-engine"]