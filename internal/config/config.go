package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv     string
	AppPort    string
	MockMode   bool
	BaseDomain string

	DatabaseURL string

	SupabaseURL            string
	SupabaseAnonKey        string
	SupabaseServiceRoleKey string

	RedisURL string

	JWTSecret    string
	JWTExpiresIn string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		AppPort:                getEnv("APP_PORT", "8080"),
		MockMode:               getBoolEnv("MOCK_MODE", true),
		BaseDomain:             getEnv("BASE_DOMAIN", "localhost"),
		DatabaseURL:            getEnv("DATABASE_URL", ""),
		SupabaseURL:            getEnv("SUPABASE_URL", ""),
		SupabaseAnonKey:        getEnv("SUPABASE_ANON_KEY", ""),
		SupabaseServiceRoleKey: getEnv("SUPABASE_SERVICE_ROLE_KEY", ""),
		RedisURL:               getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:              getEnv("JWT_SECRET", "dev-secret-change-in-production!!"),
		JWTExpiresIn:           getEnv("JWT_EXPIRES_IN", "168h"),
	}

	return cfg, nil
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	b, err := strconv.ParseBool(val)
	if err != nil {
		return fallback
	}
	return b
}
