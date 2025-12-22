package config

import (
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
)

type Config struct {
	http httpConfig
}

type httpConfig struct {
	ListenAddr      string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func FromEnv() Config {
	return Config{
		http: httpConfig{
			ListenAddr:      env.String("HTTP_LISTEN_ADDR", ":8080"),
			ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
	}
}
