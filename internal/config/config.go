package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App         AppConfig
	DB          DBConfig
	Redis       RedisConfig
	RateLimit   RateLimitConfig
	Idempotency IdempotencyConfig
	RabbitMQ    RabbitMQConfig
	Email       EmailConfig
	Admin       AdminConfig
}

type AppConfig struct {
	Port string
	Env  string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

type RedisConfig struct {
	Host string
	Port string
}

type RateLimitConfig struct {
	Enabled bool
	Limit   int
	Window  time.Duration
}

type IdempotencyConfig struct {
	Enabled bool
	TTL     time.Duration
}

type RabbitMQConfig struct {
	URL string
}

type EmailConfig struct {
	Provider    string
	SendGridKey string
	SESRegion   string
}

type AdminConfig struct {
	APIKey string
}

func Load() *Config {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("warning: no .env file found, reading from system environment")
	}

	cfg := &Config{
		App: AppConfig{
			Port: getEnvOptional("APP_PORT", "3001"),
			Env:  getEnvOptional("APP_ENV", "development"),
		},
		DB: DBConfig{
			Host:     getEnvRequired("DB_HOST"),
			Port:     getEnvRequired("DB_PORT"),
			User:     getEnvRequired("DB_USER"),
			Password: getEnvRequired("DB_PASSWORD"),
			Name:     getEnvRequired("DB_NAME"),
			SSLMode:  getEnvOptional("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host: getEnvOptional("REDIS_HOST", ""),
			Port: getEnvOptional("REDIS_PORT", ""),
		},
		RateLimit: RateLimitConfig{
			Enabled: getEnvOptionalBool("RATE_LIMIT_ENABLED", true),
			Limit:   getEnvOptionalInt("RATE_LIMIT_MAX_REQUESTS", 5),
			Window:  getEnvOptionalDuration("RATE_LIMIT_WINDOW", time.Minute),
		},
		Idempotency: IdempotencyConfig{
			Enabled: getEnvOptionalBool("IDEMPOTENCY_ENABLED", true),
			TTL:     getEnvOptionalDuration("IDEMPOTENCY_TTL", 10*time.Minute),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnvRequired("RABBITMQ_URL"),
		},
		Email: EmailConfig{
			Provider:    getEnvOptional("EMAIL_PROVIDER", "sendgrid"),
			SendGridKey: getEnvOptional("SENDGRID_API_KEY", ""),
			SESRegion:   getEnvOptional("AWS_SES_REGION", "us-east-1"),
		},
		Admin: AdminConfig{
			APIKey: getEnvOptional("ADMIN_API_KEY", ""),
		},
	}

	return cfg
}

func (c *Config) ValidateForAPI() error {
	if c.Redis.Host == "" || c.Redis.Port == "" {
		return fmt.Errorf("REDIS_HOST and REDIS_PORT are required for the API process")
	}
	if c.Admin.APIKey == "" {
		return fmt.Errorf("ADMIN_API_KEY is required for the API process")
	}
	if c.RateLimit.Enabled {
		if c.RateLimit.Limit <= 0 {
			return fmt.Errorf("RATE_LIMIT_MAX_REQUESTS must be greater than 0 when rate limiting is enabled")
		}
		if c.RateLimit.Window <= 0 {
			return fmt.Errorf("RATE_LIMIT_WINDOW must be greater than 0 when rate limiting is enabled")
		}
	}
	if c.Idempotency.Enabled && c.Idempotency.TTL <= 0 {
		return fmt.Errorf("IDEMPOTENCY_TTL must be greater than 0 when idempotency is enabled")
	}
	return nil
}

func (c *Config) ValidateForWorker() error {
	if c.Email.Provider == "" {
		return fmt.Errorf("EMAIL_PROVIDER is required for the worker process")
	}
	return nil
}

func getEnvRequired(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		panic(fmt.Sprintf("required environment variable not set: %s", key))
	}
	return value
}

func getEnvOptional(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

func getEnvOptionalInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		panic(fmt.Sprintf("invalid integer value for %s: %q", key, value))
	}

	return parsed
}

func getEnvOptionalBool(key string, defaultValue bool) bool {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		panic(fmt.Sprintf("invalid boolean value for %s: %q", key, value))
	}

	return parsed
}

func getEnvOptionalDuration(key string, defaultValue time.Duration) time.Duration {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("invalid duration value for %s: %q", key, value))
	}

	return parsed
}
