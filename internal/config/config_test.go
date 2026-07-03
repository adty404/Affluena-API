package config

import (
	"slices"
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
	t.Setenv("AUTH_RATE_LIMIT_RPS", "invalid")
	t.Setenv("AUTH_RATE_LIMIT_BURST", "-5")

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
	if cfg.AuthRateLimitRPS != 5 || cfg.AuthRateLimitBurst != 10 {
		t.Fatalf("expected default auth rate limits, got RPS %d Burst %d", cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
	}
	wantProxies := []string{"127.0.0.1/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if !slices.Equal(cfg.TrustedProxies, wantProxies) {
		t.Fatalf("expected default trusted proxies %v, got %v", wantProxies, cfg.TrustedProxies)
	}
}

func TestLoadTrustedProxiesParsing(t *testing.T) {
	defaults := []string{"127.0.0.1/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{name: "unset falls back to default", env: "", want: defaults},
		{name: "whitespace-only falls back to default", env: "   ", want: defaults},
		{name: "single entry", env: "10.1.2.0/24", want: []string{"10.1.2.0/24"}},
		{name: "trims and drops blanks", env: " 10.0.0.0/8 , , 172.16.0.0/12 ", want: []string{"10.0.0.0/8", "172.16.0.0/12"}},
		{name: "all-blank entries fall back to default", env: ",, ,", want: defaults},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRUSTED_PROXIES", tc.env)
			cfg := Load()
			if !slices.Equal(cfg.TrustedProxies, tc.want) {
				t.Fatalf("TrustedProxies = %v, want %v", cfg.TrustedProxies, tc.want)
			}
		})
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
	t.Setenv("AUTH_RATE_LIMIT_RPS", "20")
	t.Setenv("AUTH_RATE_LIMIT_BURST", "50")

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
	if cfg.AuthRateLimitRPS != 20 || cfg.AuthRateLimitBurst != 50 {
		t.Fatalf("unexpected auth rate limits, got RPS %d Burst %d", cfg.AuthRateLimitRPS, cfg.AuthRateLimitBurst)
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
			name:      "prod example placeholder is invalid",
			jwtSecret: "CHANGE_ME_GENERATE_A_RANDOM_32+_CHAR_SECRET",
			wantErr:   true,
		},
		{
			name:      "dev example placeholder is invalid",
			jwtSecret: "replace-with-a-long-random-secret-minimum-32-chars",
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

func TestConfig_Validate_ProductionDatabaseTLS(t *testing.T) {
	const secret = "this-is-a-very-long-and-secure-jwt-secret"
	const insecureURL = "postgres://u:p@postgres:5432/db?sslmode=disable"
	const secureURL = "postgres://u:p@postgres:5432/db?sslmode=require"

	tests := []struct {
		name            string
		env             string
		databaseURL     string
		allowInsecureDB bool
		wantErr         bool
	}{
		{name: "production sslmode=disable without opt-out is rejected", env: "production", databaseURL: insecureURL, wantErr: true},
		{name: "production sslmode=disable with opt-out is allowed", env: "production", databaseURL: insecureURL, allowInsecureDB: true, wantErr: false},
		{name: "production sslmode=require is allowed", env: "production", databaseURL: secureURL, wantErr: false},
		{name: "development sslmode=disable is allowed", env: "development", databaseURL: insecureURL, wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				Env:             tc.env,
				DatabaseURL:     tc.databaseURL,
				JWTSecret:       secret,
				AllowInsecureDB: tc.allowInsecureDB,
			}
			if err := cfg.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			wantInsecure := tc.env == "production" && tc.databaseURL == insecureURL
			if got := cfg.UsesInsecureProdDB(); got != wantInsecure {
				t.Errorf("UsesInsecureProdDB() = %v, want %v", got, wantInsecure)
			}
		})
	}
}
