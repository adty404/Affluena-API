package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env                         string
	HTTPAddr                    string
	DatabaseURL                 string
	JWTSecret                   string
	AccessTokenDuration         time.Duration
	RefreshTokenDuration        time.Duration
	RunMigrations               bool
	RecurringSchedulerEnabled   bool
	RecurringSchedulerInterval  time.Duration
	RecurringSchedulerBatchSize int
	CORSAllowedOrigins          string

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	AuthRateLimitRPS   int
	AuthRateLimitBurst int
}

func Load() Config {
	return Config{
		Env:                         getEnv("APP_ENV", "development"),
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://affluena_api:affluena_api@localhost:5432/affluena_api?sslmode=disable"),
		JWTSecret:                   getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenDuration:         getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenDuration:        getDurationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		RunMigrations:               getBoolEnv("RUN_MIGRATIONS", true),
		RecurringSchedulerEnabled:   getBoolEnv("RECURRING_SCHEDULER_ENABLED", true),
		RecurringSchedulerInterval:  getDurationEnv("RECURRING_SCHEDULER_INTERVAL", time.Minute),
		RecurringSchedulerBatchSize: getIntEnv("RECURRING_SCHEDULER_BATCH_SIZE", 20),
		CORSAllowedOrigins:          getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),

		SMTPHost: getEnv("SMTP_HOST", "sandbox.smtp.mailtrap.io"),
		SMTPPort: getIntEnv("SMTP_PORT", 2525),
		SMTPUser: getEnv("SMTP_USER", ""),
		SMTPPass: getEnv("SMTP_PASS", ""),
		SMTPFrom: getEnv("SMTP_FROM", "noreply@affluena.com"),

		AuthRateLimitRPS:   getPositiveIntEnv("AUTH_RATE_LIMIT_RPS", 5),
		AuthRateLimitBurst: getPositiveIntEnv("AUTH_RATE_LIMIT_BURST", 10),
	}
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getBoolEnv(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getIntEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func getPositiveIntEnv(key string, fallback int) int {
	val := getIntEnv(key, fallback)
	if val <= 0 {
		return fallback
	}
	return val
}

const minJWTSecretLength = 32

func (c Config) Validate() error {
	if c.JWTSecret == "" || c.JWTSecret == "change-me-in-production" {
		return errors.New("JWT_SECRET must be set and not be the default value")
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long", minJWTSecretLength)
	}
	return nil
}
