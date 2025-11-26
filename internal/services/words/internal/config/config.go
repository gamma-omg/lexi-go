package config

import (
	"fmt"
	"os"
)

type Config struct {
	ListenAddr  string
	AuthSecret  string
	TagsMaxKeys int64
	TagsMaxCost int64
	DB          dbConfig
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
		ListenAddr:  getEnvString("LISTEN_ADDR", ":8080"),
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
