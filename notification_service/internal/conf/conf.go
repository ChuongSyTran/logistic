package conf

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
}

type ServerConfig struct {
	GrpcPort string `env:"NOTIFICATION_SERVICE_PORT" env-default:"9006"`
}

type DatabaseConfig struct {
	Driver   string `env:"NOTIFICATION_SERVICE_DB_DRIVER" env-default:"postgres"`
	User     string `env:"NOTIFICATION_DB_USER" env-required:"true"`
	Password string `env:"NOTIFICATION_DB_PASSWORD" env-required:"true"`
	Host     string `env:"NOTIFICATION_DB_HOST" env-required:"true"`
	Port     string `env:"NOTIFICATION_DB_PORT" env-required:"true"`
	DBName   string `env:"NOTIFICATION_DB_NAME" env-required:"true"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" env-default:"redis"`
	Port     string `env:"REDIS_PORT" env-default:"6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"NOTIFICATION_REDIS_DB" env-default:"2"`
	Prefix   string `env:"NOTIFICATION_REDIS_PREFIX" env-default:"notif"`
	Enabled  bool   `env:"NOTIFICATION_REDIS_ENABLED" env-default:"true"`
}

type RabbitMQConfig struct {
	Host     string `env:"RABBITMQ_HOST" env-default:"rabbitmq"`
	Port     string `env:"RABBITMQ_PORT" env-default:"5672"`
	User     string `env:"RABBITMQ_USER" env-default:"guest"`
	Password string `env:"RABBITMQ_PASSWORD" env-default:"guest"`
	VHost    string `env:"RABBITMQ_VHOST" env-default:"/"`
	Exchange string `env:"RABBITMQ_EXCHANGE" env-default:"logistic.events"`
	Queue    string `env:"NOTIFICATION_MQ_QUEUE" env-default:"notification.events"`

	BindingKeys string `env:"NOTIFICATION_MQ_BINDINGS" env-default:"matching.#"`
	Prefetch    int    `env:"NOTIFICATION_MQ_PREFETCH" env-default:"20"`
	Enabled     bool   `env:"NOTIFICATION_MQ_ENABLED" env-default:"true"`
}

func (db *DatabaseConfig) GetDataSource() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadEnv(cfg)
	return cfg, err
}