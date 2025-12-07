package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AuthSecret  string
	TagsMaxKeys int64
	TagsMaxCost int64
	DB          dbConfig
	Http        httpConfig
}

type dbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type httpConfig struct {
	ListenAddr      string
	IdleTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func FromEnv() Config {
	return Config{
		AuthSecret:  os.Getenv("AUTH_SECRET"),
		TagsMaxKeys: getEnvInt64("TAGS_CACHE_KEYS", 10000),
		TagsMaxCost: getEnvInt64("TAGS_CACHE_COST", 10000),
		DB: dbConfig{
			Host:     getEnvString("DB_HOST", "localhost"),
			Port:     getEnvString("DB_PORT", "5432"),
			User:     getEnvString("DB_USER", "postgres"),
			Password: getEnvString("DB_PASSWORD", "password"),
			Name:     getEnvString("DB_NAME", "words_service"),
		},
		Http: httpConfig{
			ListenAddr:      getEnvString("HTTP_LISTEN_ADDR", ":8080"),
			IdleTimeout:     getEnvDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ReadTimeout:     getEnvDuration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    getEnvDuration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: getEnvDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
	}
}

func getEnvString(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists {
		return val
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if valStr, exists := os.LookupEnv(key); exists {
		var val int64
		_, err := fmt.Sscanf(valStr, "%d", &val)
		if err == nil {
			return val
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if valStr, exists := os.LookupEnv(key); exists {
		val, err := time.ParseDuration(valStr)
		if err == nil {
			return val
		}
	}
	return defaultVal
}
