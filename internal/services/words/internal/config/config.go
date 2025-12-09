package config

import (
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
)

type Config struct {
	AuthSecret  string
	TagsMaxKeys int64
	TagsMaxCost int64
	DB          dbConfig
	Http        httpConfig
	Image       imageConfig
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

type imageConfig struct {
	Endpoint  string
	FieldName string
	FileName  string
}

func FromEnv() Config {
	return Config{
		AuthSecret:  env.RequireString("AUTH_SECRET"),
		TagsMaxKeys: env.Int64("TAGS_CACHE_KEYS", 10000),
		TagsMaxCost: env.Int64("TAGS_CACHE_COST", 10000),
		DB: dbConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "postgres"),
			Password: env.String("DB_PASSWORD", "password"),
			Name:     env.String("DB_NAME", "words_service"),
		},
		Http: httpConfig{
			ListenAddr:      env.String("HTTP_LISTEN_ADDR", ":8080"),
			IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Image: imageConfig{
			Endpoint:  env.String("IMAGE_ENDPOINT", "http://localhost:9999/"),
			FieldName: env.String("IMAGE_FIELD_NAME", "image"),
			FileName:  env.String("IMAGE_FILE_NAME", "image.jpg"),
		},
	}
}
