package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// AllowInsecureDB opts out of the production sslmode=disable guard. It exists
	// for deployments where Postgres runs on the same trusted host/Docker network
	// (no TLS terminator), where cleartext traffic never leaves the host. Default
	// false: production refuses to boot with sslmode=disable unless this is set.
	AllowInsecureDB bool

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
		AllowInsecureDB:             getBoolEnv("ALLOW_INSECURE_DB", false),

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
	if c.JWTSecret == "" || isPlaceholderSecret(c.JWTSecret) {
		return errors.New("JWT_SECRET must be set to a real random value, not a default/placeholder")
	}
	if len(c.JWTSecret) < minJWTSecretLength {
		return fmt.Errorf("JWT_SECRET must be at least %d characters long", minJWTSecretLength)
	}
	// In production the database connection must be encrypted; refuse to boot
	// with sslmode=disable so credentials and row data are not sent in cleartext.
	// Operators running Postgres on the same trusted host/Docker network (where
	// traffic never leaves the machine) can set ALLOW_INSECURE_DB=true to opt out.
	if c.UsesInsecureProdDB() && !c.AllowInsecureDB {
		return errors.New("DATABASE_URL must not use sslmode=disable in production; use sslmode=require/verify-full, or set ALLOW_INSECURE_DB=true for a trusted same-host database")
	}
	return nil
}

// UsesInsecureProdDB reports whether the running config is a production
// deployment whose database connection is unencrypted (sslmode=disable).
func (c Config) UsesInsecureProdDB() bool {
	return c.Env == "production" && strings.Contains(c.DatabaseURL, "sslmode=disable")
}

// isPlaceholderSecret flags the example/template JWT secrets shipped in the
// .env*.example files so an operator who forgets to replace one fails fast
// instead of running with a publicly-known secret.
func isPlaceholderSecret(secret string) bool {
	lower := strings.ToLower(secret)
	for _, marker := range []string{"change-me", "change_me", "replace-with"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
