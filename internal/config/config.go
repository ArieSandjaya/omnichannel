package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppEnv    string
	AppPort   string
	AppURL    string
	JWTSecret string

	DB struct {
		Host     string
		Port     string
		User     string
		Password string
		Name     string
		SSLMode  string
	}

	Redis struct {
		Addr     string
		Password string
		DB       int
	}

	Xendit struct {
		SecretKey         string
		WebhookToken      string
		QRISCallbackURL   string
		VACallbackURL     string
		QRISExpirySeconds int
		VAExpiryHours     int
		SupportedVABanks  []string
	}

	Biteship struct {
		APIKey        string
		APIURL        string
		WebhookSecret string
	}
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("config: no .env file found, reading from environment: %v", err)
	}

	c := &Config{}

	c.AppEnv = viper.GetString("APP_ENV")
	if c.AppEnv == "" {
		c.AppEnv = "development"
	}
	c.AppPort = viper.GetString("APP_PORT")
	if c.AppPort == "" {
		c.AppPort = "8080"
	}
	c.AppURL = viper.GetString("APP_URL")
	c.JWTSecret = viper.GetString("JWT_SECRET")

	c.DB.Host = viper.GetString("DB_HOST")
	c.DB.Port = viper.GetString("DB_PORT")
	if c.DB.Port == "" {
		c.DB.Port = "5432"
	}
	c.DB.User = viper.GetString("DB_USER")
	c.DB.Password = viper.GetString("DB_PASS")
	c.DB.Name = viper.GetString("DB_NAME")
	c.DB.SSLMode = viper.GetString("DB_SSL_MODE")
	if c.DB.SSLMode == "" {
		c.DB.SSLMode = "disable"
	}

	c.Redis.Addr = viper.GetString("REDIS_ADDR")
	if c.Redis.Addr == "" {
		c.Redis.Addr = "localhost:6379"
	}
	c.Redis.Password = viper.GetString("REDIS_PASSWORD")
	c.Redis.DB = viper.GetInt("REDIS_DB")

	c.Xendit.SecretKey = viper.GetString("XENDIT_SECRET_KEY")
	c.Xendit.WebhookToken = viper.GetString("XENDIT_WEBHOOK_TOKEN")
	c.Xendit.QRISCallbackURL = viper.GetString("XENDIT_QRIS_CALLBACK_URL")
	c.Xendit.VACallbackURL = viper.GetString("XENDIT_VA_CALLBACK_URL")
	c.Xendit.QRISExpirySeconds = viper.GetInt("QRIS_EXPIRY_SECONDS")
	if c.Xendit.QRISExpirySeconds == 0 {
		c.Xendit.QRISExpirySeconds = 3600
	}
	c.Xendit.VAExpiryHours = viper.GetInt("VA_EXPIRY_HOURS")
	if c.Xendit.VAExpiryHours == 0 {
		c.Xendit.VAExpiryHours = 24
	}
	banks := viper.GetString("XENDIT_VA_BANKS")
	if banks == "" {
		banks = "BCA,BNI,BRI,MANDIRI"
	}
	c.Xendit.SupportedVABanks = strings.Split(banks, ",")

	c.Biteship.APIKey = viper.GetString("BITESHIP_API_KEY")
	c.Biteship.APIURL = viper.GetString("BITESHIP_API_URL")
	if c.Biteship.APIURL == "" {
		c.Biteship.APIURL = "https://api.biteship.com/v1"
	}
	c.Biteship.WebhookSecret = viper.GetString("BITESHIP_WEBHOOK_SECRET")

	return c
}

func (c *Config) DBConnString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.DB.User, c.DB.Password, c.DB.Host, c.DB.Port, c.DB.Name, c.DB.SSLMode)
}
