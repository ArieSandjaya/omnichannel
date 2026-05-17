package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv      string
	AppPort     string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		slog.Warn("no .env file, reading from environment")
	}
	return &Config{
		AppEnv:      getEnv("APP_ENV", "development"),
		AppPort:     getEnv("APP_PORT", "8080"),
		DatabaseURL: mustGetEnv("DATABASE_URL"),
		RedisURL:    getEnv("REDIS_URL", ""),
		JWTSecret:   mustGetEnv("JWT_SECRET"),
	}
}

func getEnv(key, fallback string) string {
	if v := viper.GetString(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := viper.GetString(key)
	if v == "" {
		slog.Error("required env var not set", "key", key)
		os.Exit(1)
	}
	return v
}
