package config

import "os"

type Config struct {
	ListenAddr string
	AuthSecret string
	DB         dbConfig
}

type dbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func FromEnv() Config {
	return Config{
		ListenAddr: getEnvOrDefault("LISTEN_ADDR", ":8080"),
		AuthSecret: os.Getenv("AUTH_SECRET"),
		DB: dbConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "password"),
			Name:     getEnvOrDefault("DB_NAME", "words_service"),
		},
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}
