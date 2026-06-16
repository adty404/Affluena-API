package config

import (
	"testing"
	"time"
)

func TestLoadUsesDefaultsWhenEnvIsMissingOrInvalid(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("ACCESS_TOKEN_TTL", "bad-duration")
	t.Setenv("REFRESH_TOKEN_TTL", "bad-duration")
	t.Setenv("RUN_MIGRATIONS", "not-bool")
	t.Setenv("RECURRING_SCHEDULER_ENABLED", "not-bool")
	t.Setenv("RECURRING_SCHEDULER_INTERVAL", "bad-duration")
	t.Setenv("RECURRING_SCHEDULER_BATCH_SIZE", "not-int")

	cfg := Load()

	if cfg.Env != "development" || cfg.HTTPAddr != ":8080" {
		t.Fatalf("expected default env/http addr, got %+v", cfg)
	}
	if cfg.AccessTokenDuration != 15*time.Minute {
		t.Fatalf("expected default access token ttl, got %s", cfg.AccessTokenDuration)
	}
	if cfg.RefreshTokenDuration != 30*24*time.Hour {
		t.Fatalf("expected default refresh token ttl, got %s", cfg.RefreshTokenDuration)
	}
	if !cfg.RunMigrations || !cfg.RecurringSchedulerEnabled || cfg.RecurringSchedulerBatchSize != 20 {
		t.Fatalf("expected default bool/int values, got %+v", cfg)
	}
}

func TestLoadParsesExplicitEnv(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("ACCESS_TOKEN_TTL", "2m")
	t.Setenv("REFRESH_TOKEN_TTL", "48h")
	t.Setenv("RUN_MIGRATIONS", "false")
	t.Setenv("RECURRING_SCHEDULER_ENABLED", "false")
	t.Setenv("RECURRING_SCHEDULER_INTERVAL", "5s")
	t.Setenv("RECURRING_SCHEDULER_BATCH_SIZE", "7")

	cfg := Load()

	if cfg.Env != "test" || cfg.HTTPAddr != ":9090" || cfg.DatabaseURL != "postgres://example" || cfg.JWTSecret != "secret" {
		t.Fatalf("unexpected string config values: %+v", cfg)
	}
	if cfg.AccessTokenDuration != 2*time.Minute || cfg.RefreshTokenDuration != 48*time.Hour || cfg.RecurringSchedulerInterval != 5*time.Second {
		t.Fatalf("unexpected duration config values: %+v", cfg)
	}
	if cfg.RunMigrations || cfg.RecurringSchedulerEnabled || cfg.RecurringSchedulerBatchSize != 7 {
		t.Fatalf("unexpected bool/int config values: %+v", cfg)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name      string
		jwtSecret string
		wantErr   bool
	}{
		{
			name:      "valid config with secure JWT secret",
			jwtSecret: "this-is-a-very-long-and-secure-jwt-secret",
			wantErr:   false,
		},
		{
			name:      "empty JWT secret is invalid",
			jwtSecret: "",
			wantErr:   true,
		},
		{
			name:      "default JWT secret is invalid",
			jwtSecret: "change-me-in-production",
			wantErr:   true,
		},
		{
			name:      "too short JWT secret is invalid",
			jwtSecret: "short-secret",
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{JWTSecret: tc.jwtSecret}
			err := cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
