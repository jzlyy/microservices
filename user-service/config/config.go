package config

import (
	"os"
)

type Config struct {
	DBUser     string
	DBPassword string
	DBHost     string
	DBPort     string
	DBName     string
	JWTSecret  string
}

func LoadConfig() *Config {
	return &Config{
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "qzhufuchengz"),
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBName:     getEnv("DB_NAME", "ecommerce"),
		JWTSecret:  getEnv("JWT_SECRET", "G9mCQ19ogTkuWQY9jH2wGZASuGi/JrhstQaZy4k/01o="),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
