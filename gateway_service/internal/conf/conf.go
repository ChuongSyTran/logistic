package conf

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server    ServerConfig
	Auth      AuthConfig
	Upstreams UpstreamConfig
	CORS      CORSConfig
}

type ServerConfig struct {
	Port         string        `env:"GATEWAY_PORT" env-required:"true"`
	IsProduction bool          `env:"GLOBAL_IS_PRODUCTION" env-default:"false"`
	ReadTimeout  time.Duration `env:"GATEWAY_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout time.Duration `env:"GATEWAY_WRITE_TIMEOUT" env-default:"30s"`

	IdleTimeout time.Duration `env:"GATEWAY_IDLE_TIMEOUT" env-default:"60s"`
}

type AuthConfig struct {
	PublicKey string `env:"GATEWAY_JWT_PUBLIC_KEY" env-required:"true"`

	PreviousPublicKey string `env:"GATEWAY_JWT_PREVIOUS_PUBLIC_KEY" env-default:""`
}

type UpstreamConfig struct {
	Auth         string `env:"GATEWAY_AUTH_GRPC_ADDR" env-default:"auth-service:9001"`
	Media        string `env:"GATEWAY_MEDIA_GRPC_ADDR" env-default:"media-service:9002"`
	Matching     string `env:"GATEWAY_MATCHING_GRPC_ADDR" env-default:"matching-service:9003"`
	User         string `env:"GATEWAY_USER_GRPC_ADDR" env-default:"user-service:9004"`
	Vehicle      string `env:"GATEWAY_VEHICLE_GRPC_ADDR" env-default:"vehicle-service:9005"`
	Notification string `env:"GATEWAY_NOTIFICATION_GRPC_ADDR" env-default:"notification-service:9006"`
	Kyc          string `env:"GATEWAY_KYC_GRPC_ADDR" env-default:"kyc-service:9009"`

	CallTimeout time.Duration `env:"GATEWAY_GRPC_TIMEOUT" env-default:"5s"`

	LocationTimeout time.Duration `env:"GATEWAY_GRPC_LOCATION_TIMEOUT" env-default:"2s"`

	MaxConcurrentPerUpstream int `env:"GATEWAY_MAX_CONCURRENT_PER_UPSTREAM" env-default:"256"`
}

type CORSConfig struct {
	AllowOrigins []string `env:"GATEWAY_CORS_ORIGINS" env-separator:"," env-default:"http://localhost:3000,http://localhost:8080"`
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("gateway conf: %w", err)
	}
	return cfg, nil
}