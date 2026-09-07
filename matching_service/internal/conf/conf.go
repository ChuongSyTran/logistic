package conf

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server         ServerConfig
	MasterDatabase MasterDatabaseConfig
	SlaveDatabase  SlaveDatabaseConfig
	NatConfig      NatConfig
	KafkaConfig    KafkaConfig
	RabbitMQ       RabbitMQConfig
	WalletService  WalletServiceConfig
	VehicleService VehicleServiceConfig
}

type ServerConfig struct {
	GrpcPort     string `env:"MATCHING_SERVICE_GRPC_PORT" env-required:"true"`
	IsProduction bool   `env:"GLOBAL_IS_PRODUCTION" env-default:"false"`
}

type MasterDatabaseConfig struct {
	Driver   string `env:"MATCHING_SERVICE_DB_DRIVER" env-required:"true"`
	User     string `env:"MATCHING_DB_USER" env-required:"true"`
	Password string `env:"MATCHING_DB_PASSWORD" env-required:"true"`
	Host     string `env:"MATCHING_DB_MASTER_HOST" env-required:"true"`
	Port     string `env:"MATCHING_DB_MASTER_PORT" env-required:"true"`
	DBName   string `env:"MATCHING_DB_NAME" env-required:"true"`
}

type SlaveDatabaseConfig struct {
	Driver   string `env:"MATCHING_SERVICE_DB_DRIVER" env-required:"true"`
	User     string `env:"MATCHING_DB_USER" env-required:"true"`
	Password string `env:"MATCHING_DB_PASSWORD" env-required:"true"`
	Host     string `env:"MATCHING_DB_SLAVE_HOST" env-required:"true"`
	Port     string `env:"MATCHING_DB_SLAVE_PORT" env-required:"true"`
	DBName   string `env:"MATCHING_DB_NAME" env-required:"true"`
}

type NatConfig struct {
	Host string `env:"NATS_HOST" env-required:"true"`
	Port string `env:"NATS_PORT" env-required:"true"`
}

type KafkaConfig struct {
	ClusterId string `env:"KAFKA_KRAFT_CLUSTER_ID" env-required:"true"`
	Brokers   string `env:"KAFKA_BROKERS" env-required:"true"`
}

type WalletServiceConfig struct {
	GrpcAddr string `env:"MATCHING_WALLET_GRPC_ADDR" env-default:"wallet-service:9007"`
}

func (db *MasterDatabaseConfig) GetDataSource() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}

func (db *SlaveDatabaseConfig) GetDataSource() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadEnv(cfg)
	return cfg, err
}

type RabbitMQConfig struct {
	Host     string `env:"RABBITMQ_HOST" env-default:"rabbitmq"`
	Port     string `env:"RABBITMQ_PORT" env-default:"5672"`
	User     string `env:"RABBITMQ_USER" env-default:"guest"`
	Password string `env:"RABBITMQ_PASSWORD" env-default:"guest"`
	VHost    string `env:"RABBITMQ_VHOST" env-default:"/"`
	Exchange string `env:"RABBITMQ_EXCHANGE" env-default:"logistic.events"`
	Enabled  bool   `env:"MATCHING_MQ_ENABLED" env-default:"true"`
}

type VehicleServiceConfig struct {
	GrpcAddr string `env:"MATCHING_VEHICLE_GRPC_ADDR" env-default:"vehicle-service:9005"`
}