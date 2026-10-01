// Package config 加载应用配置：configs/config.yaml 提供默认值，
// 环境变量（前缀 MK_，层级用下划线连接，如 MK_MYSQL_PASSWORD）覆盖其中的值。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	App   AppConfig   `mapstructure:"app"`
	Log   LogConfig   `mapstructure:"log"`
	MySQL MySQLConfig `mapstructure:"mysql"`
	Redis RedisConfig `mapstructure:"redis"`
	S3    S3Config    `mapstructure:"s3"`
	Auth  AuthConfig  `mapstructure:"auth"`
	Mail  MailConfig  `mapstructure:"mail"`
}

type AppConfig struct {
	Name            string        `mapstructure:"name"`
	Env             string        `mapstructure:"env"` // development / test / production
	Port            int           `mapstructure:"port"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	CORSOrigins     []string      `mapstructure:"cors_origins"`
	// PublicURL 是前端站点地址，用于生成邮件中的链接
	PublicURL string `mapstructure:"public_url"`
}

type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug / info / warn / error
	Format string `mapstructure:"format"` // console / json
}

type MySQLConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Database        string        `mapstructure:"database"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN 返回 go-sql-driver/mysql 格式的连接串。时间统一按 UTC 读写。
func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&collation=utf8mb4_0900_ai_ci&parseTime=true&loc=UTC",
		m.User, m.Password, m.Host, m.Port, m.Database)
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type S3Config struct {
	Endpoint      string `mapstructure:"endpoint"`
	Region        string `mapstructure:"region"`
	AccessKey     string `mapstructure:"access_key"`
	SecretKey     string `mapstructure:"secret_key"`
	PublicBucket  string `mapstructure:"public_bucket"`
	PrivateBucket string `mapstructure:"private_bucket"`
	UsePathStyle  bool   `mapstructure:"use_path_style"`
}

type AuthConfig struct {
	JWTSecret        string        `mapstructure:"jwt_secret"`
	AccessTTL        time.Duration `mapstructure:"access_ttl"`
	RefreshTTL       time.Duration `mapstructure:"refresh_ttl"`
	RememberTTL      time.Duration `mapstructure:"remember_ttl"`
	RefreshCookie    string        `mapstructure:"refresh_cookie"`
	CookieSecure     bool          `mapstructure:"cookie_secure"`
	BcryptCost       int           `mapstructure:"bcrypt_cost"`
	VerifyEmailTTL   time.Duration `mapstructure:"verify_email_ttl"`
	ResetPasswordTTL time.Duration `mapstructure:"reset_password_ttl"`
}

type MailConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
	// TLS 为 false 时使用明文连接，仅用于本地 Mailpit
	TLS bool `mapstructure:"tls"`
}

func (c *Config) IsProduction() bool { return c.App.Env == "production" }

// Load 读取配置。dir 为 config.yaml 所在目录；当前目录存在 .env 时先加载（仅开发环境使用）。
func Load(dir string) (*Config, error) {
	_ = godotenv.Load() // .env 不存在时忽略

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	v.SetEnvPrefix("MK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	var missing []string
	if c.MySQL.Password == "" {
		missing = append(missing, "MK_MYSQL_PASSWORD")
	}
	if c.Redis.Password == "" {
		missing = append(missing, "MK_REDIS_PASSWORD")
	}
	if len(c.Auth.JWTSecret) < 32 {
		missing = append(missing, "MK_AUTH_JWT_SECRET (at least 32 characters)")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}
