package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port                  string
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBSSLMode             string
	JWTSecret             string
	RazorpayKeyID         string
	RazorpayKeySecret     string
	RazorpayWebhookSecret string
	MaxOTPRatePerMinute   int
}

func Load() *Config {
	return &Config{
		Port:                  getEnv("PORT", "8080"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "spider"),
		DBPassword:            getEnv("DB_PASSWORD", "spider_secure_pass"),
		DBName:                getEnv("DB_NAME", "spider_freight"),
		DBSSLMode:             getEnv("DB_SSLMODE", "disable"),
		JWTSecret:             getEnv("JWT_SECRET", "super_spider_jwt_secret_for_dev_change_in_prod_123456"),
		RazorpayKeyID:         getEnv("RAZORPAY_KEY_ID", "rzp_test_dummy"),
		RazorpayKeySecret:     getEnv("RAZORPAY_KEY_SECRET", "rzp_test_secret_dummy"),
		RazorpayWebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", "rzp_webhook_secret_dummy"),
		MaxOTPRatePerMinute:   getEnvAsInt("MAX_OTP_RATE_PER_MINUTE", 5),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}
