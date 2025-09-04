# QTest Service - Intelligent Automated Testing Suite
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o qtest .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/qtest .

EXPOSE 8091

CMD ["./qtest"]