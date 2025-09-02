package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Audit    AuditConfig    `mapstructure:"audit"`
}

type ServerConfig struct {
	Port                    int    `mapstructure:"port"`
	MetricsPort             int    `mapstructure:"metrics_port"`
	GracefulShutdownTimeout int    `mapstructure:"graceful_shutdown_timeout"`
	Environment             string `mapstructure:"environment"`
}

type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	Database       string `mapstructure:"database"`
	User           string `mapstructure:"user"`
	Password       string `mapstructure:"password"`
	SSLMode        string `mapstructure:"ssl_mode"`
	MaxConnections int    `mapstructure:"max_connections"`
	MaxIdleConns   int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	Enabled  bool   `mapstructure:"enabled"`
}

type LLMConfig struct {
	DefaultProvider string                 `mapstructure:"default_provider"`
	Timeout         int                    `mapstructure:"timeout"`
	MaxRetries      int                    `mapstructure:"max_retries"`
	Providers       map[string]interface{} `mapstructure:"providers"`
}

type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Endpoint     string  `mapstructure:"endpoint"`
	SamplingRate float64 `mapstructure:"sampling_rate"`
	ServiceName  string  `mapstructure:"service_name"`
}

type AuditConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	LogPath string `mapstructure:"log_path"`
	Level   string `mapstructure:"level"`
}

// Load loads configuration from environment variables and config files
func Load(serviceName string) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.metrics_port", 9090)
	v.SetDefault("server.graceful_shutdown_timeout", 30)
	v.SetDefault("server.environment", "development")

	v.SetDefault("database.host", "postgres-rw.quantumlayer.svc.cluster.local")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.database", "quantumlayer")
	v.SetDefault("database.ssl_mode", "require")
	v.SetDefault("database.max_connections", 100)
	v.SetDefault("database.max_idle_conns", 10)

	v.SetDefault("redis.host", "redis-master.quantumlayer.svc.cluster.local")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.enabled", true)

	v.SetDefault("llm.default_provider", "aws-bedrock")
	v.SetDefault("llm.timeout", 60)
	v.SetDefault("llm.max_retries", 3)

	v.SetDefault("tracing.enabled", true)
	v.SetDefault("tracing.endpoint", "jaeger-collector.istio-system.svc.cluster.local:14268")
	v.SetDefault("tracing.sampling_rate", 0.1)
	v.SetDefault("tracing.service_name", serviceName)

	v.SetDefault("audit.enabled", true)
	v.SetDefault("audit.log_path", "/var/log/audit")
	v.SetDefault("audit.level", "info")

	// Read from environment variables
	v.SetEnvPrefix(strings.ToUpper(strings.ReplaceAll(serviceName, "-", "_")))
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read from config file if exists
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "/app/config"
	}
	
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)
	v.AddConfigPath(".")
	
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found; use defaults and environment
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Override with specific environment variables
	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		config.Database.Host = host
	}
	if port := os.Getenv("POSTGRES_PORT"); port != "" {
		fmt.Sscanf(port, "%d", &config.Database.Port)
	}
	if user := os.Getenv("POSTGRES_USER"); user != "" {
		config.Database.User = user
	}
	if password := os.Getenv("POSTGRES_PASSWORD"); password != "" {
		config.Database.Password = password
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Redis.Host = redisHost
	}

	return &config, nil
}