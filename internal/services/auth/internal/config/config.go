package config

import (
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
)

type Config struct {
	HTTP httpConfig
	JWT  jwtConfig
}

type httpConfig struct {
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type jwtConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func FromEnv() Config {
	return Config{
		HTTP: httpConfig{
			ListenAddr:      env.String("HTTP_LISTEN_ADDR", ":8080"),
			ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		JWT: jwtConfig{
			AccessSecret:  env.RequireString("JWT_ACCESS_SECRET"),
			RefreshSecret: env.RequireString("JWT_REFRESH_SECRET"),
			AccessTTL:     env.Duration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    env.Duration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
	}
}
