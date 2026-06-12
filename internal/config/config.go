package config

import (
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
}

func Load() Config {
	return Config{
		Env:                         getEnv("APP_ENV", "development"),
		HTTPAddr:                    getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:                 getEnv("DATABASE_URL", "postgres://affluena:affluena@localhost:5432/affluena?sslmode=disable"),
		JWTSecret:                   getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenDuration:         getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenDuration:        getDurationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		RunMigrations:               getBoolEnv("RUN_MIGRATIONS", true),
		RecurringSchedulerEnabled:   getBoolEnv("RECURRING_SCHEDULER_ENABLED", true),
		RecurringSchedulerInterval:  getDurationEnv("RECURRING_SCHEDULER_INTERVAL", time.Minute),
		RecurringSchedulerBatchSize: getIntEnv("RECURRING_SCHEDULER_BATCH_SIZE", 20),
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
