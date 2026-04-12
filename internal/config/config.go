package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	RabbitMQ RabbitMQConfig
	Email    EmailConfig
	Admin    AdminConfig
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

	return &Config{
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
			Host: getEnvRequired("REDIS_HOST"),
			Port: getEnvRequired("REDIS_PORT"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnvRequired("RABBITMQ_URL"),
		},
		Email: EmailConfig{
			Provider:    getEnvRequired("EMAIL_PROVIDER"),
			SendGridKey: getEnvOptional("SENDGRID_API_KEY", ""),
			SESRegion:   getEnvOptional("AWS_SES_REGION", "us-east-1"),
		},
		Admin: AdminConfig{
			APIKey: getEnvRequired("ADMIN_API_KEY"),
		},
	}
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
