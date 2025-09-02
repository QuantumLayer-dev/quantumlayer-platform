module github.com/QuantumLayer-dev/quantumlayer-platform/packages/agent-ensemble

go 1.21

require (
	github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared v0.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.4.0
	github.com/jackc/pgx/v5 v5.5.0
	github.com/nats-io/nats.go v1.31.0
	github.com/qdrant/go-client v1.7.0
	github.com/redis/go-redis/v9 v9.3.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/viper v1.17.0
	go.opentelemetry.io/otel v1.19.0
	go.opentelemetry.io/otel/trace v1.19.0
)

replace github.com/QuantumLayer-dev/quantumlayer-platform/packages/shared => ../shared