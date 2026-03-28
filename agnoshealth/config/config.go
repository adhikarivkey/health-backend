package config

import (
	"log"
	"os"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	Port       string
	HISBaseURL string
	GinMode    string
}

var AppConfig *Config

func Load() {
	AppConfig = &Config{
		DBHost:     getEnv("DB_HOST", ""),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", ""),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		Port:       getEnv("PORT", "8000"),
		HISBaseURL: getEnv("HIS_BASE_URL", "https://hospital-a.api.co.th"),
		GinMode:    getEnv("GIN_MODE", "debug"),
	}

	validate()
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// validate checks that all required environment variables are set
func validate() {
	if AppConfig.DBHost == "" ||
		AppConfig.DBUser == "" ||
		AppConfig.DBPassword == "" ||
		AppConfig.DBName == "" ||
		AppConfig.JWTSecret == "" {

		log.Fatal("Missing required environment variables")
	}
}
