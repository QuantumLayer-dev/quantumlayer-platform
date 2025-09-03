FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download || go mod tidy

# Copy source
COPY . .

# Build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o workflow-api main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/workflow-api .

EXPOSE 8080

CMD ["./workflow-api"]