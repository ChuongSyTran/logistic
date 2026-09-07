package conf

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Google    GoogleConfig
	JWT       JWTConfig
	Bootstrap BootstrapConfig
}

// Phải khai cả email lẫn mật khẩu mới chạy; không đặt mặc định vì admin có mật
// khẩu đoán được còn tệ hơn không có admin.
type BootstrapConfig struct {
	AdminEmail    string `env:"AUTH_SERVICE_BOOTSTRAP_ADMIN_EMAIL"`
	AdminPassword string `env:"AUTH_SERVICE_BOOTSTRAP_ADMIN_PASSWORD"`
	AdminFullName string `env:"AUTH_SERVICE_BOOTSTRAP_ADMIN_NAME" env-default:"Quản trị hệ thống"`
}

func (b BootstrapConfig) Enabled() bool {
	return b.AdminEmail != "" && b.AdminPassword != ""
}

type ServerConfig struct {
	GrpcPort     string `env:"AUTH_SERVICE_GRPC_PORT" env-required:"true"`
	IsProduction bool   `env:"GLOBAL_IS_PRODUCTION" env-default:"false"`
}

const (
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
)

type DatabaseConfig struct {
	Driver   string `env:"AUTH_SERVICE_DB_DRIVER" env-required:"true"`
	User     string `env:"AUTH_SERVICE_DB_USER" env-required:"true"`
	Password string `env:"AUTH_SERVICE_DB_PASSWORD" env-required:"true"`
	Host     string `env:"AUTH_SERVICE_DB_HOST" env-required:"true"`
	Port     string `env:"AUTH_SERVICE_DB_PORT" env-required:"true"`
	DBName   string `env:"AUTH_SERVICE_DB_NAME" env-required:"true"`
}

type GoogleConfig struct {
	ClientID     string `env:"AUTH_SERVICE_GOOGLE_CLIENT_ID" env-required:"true"`
	ClientSecret string `env:"AUTH_SERVICE_GOOGLE_CLIENT_SECRET" env-required:"true"`
	RedirectURL  string `env:"AUTH_SERVICE_GOOGLE_REDIRECT_URL" env-required:"true"`
}

type JWTConfig struct {
	PrivateKey string        `env:"AUTH_SERVICE_JWT_PRIVATE_KEY" env-required:"true"`
	AccessTTL  time.Duration `env:"AUTH_SERVICE_JWT_ACCESS_TTL" env-default:"15m"`
	RefreshTTL time.Duration `env:"AUTH_SERVICE_JWT_REFRESH_TTL" env-default:"168h"`
}

func (db *DatabaseConfig) GetDataSource() string {
	if db.Driver == DriverMySQL {
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC",
			db.User, db.Password, db.Host, db.Port, db.DBName)
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		db.Host, db.Port, db.User, db.Password, db.DBName)
}

func LoadConfig() (*Config, error) {
	cfg := &Config{}
	err := cleanenv.ReadEnv(cfg)
	return cfg, err
}
