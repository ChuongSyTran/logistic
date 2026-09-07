package conf

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Kafka         KafkaConfig
	ElasticSearch ElasticSearchConfig
	Telemetry     TelemetryConfig
}

type ServerConfig struct {
	GrpcPort string `env:"WALLET_SERVICE_GRPC_PORT" env-default:"9007"`
}

type DatabaseConfig struct {
	DSN string `env:"WALLET_SERVICE_DB_DSN" env-required:"true"`
}

type KafkaConfig struct {
	Brokers string `env:"WALLET_SERVICE_KAFKA_BROKERS" env-default:"localhost:9092"`
}

type ElasticSearchConfig struct {
	Addresses string `env:"WALLET_SERVICE_ES_ADDRESSES" env-default:"http://localhost:9200"`
	Username  string `env:"WALLET_SERVICE_ES_USERNAME" env-default:"elastic"`
	Password  string `env:"WALLET_SERVICE_ES_PASSWORD"`
}

type TelemetryConfig struct {
	OtlpEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadEnv(cfg)
	return cfg, err
}
