package config

import (
	"net/url"
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
)

type Config struct {
	AuthSecret string
	HTTP       httpConfig
	ImageStore imageConfig
}

type httpConfig struct {
	ListenAddr      string
	ListenPort      int
	IdleTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type imageConfig struct {
	Root      string
	ServeRoot *url.URL
	MaxSize   int64
	MaxWidth  int
	MaxHeight int
}

func FromEnv() Config {
	return Config{
		AuthSecret: env.RequireString("AUTH_SECRET"),
		HTTP: httpConfig{
			ListenAddr:      env.String("HTTP_LISTEN_ADDR", ""),
			ListenPort:      env.Int("HTTP_LISTEN_PORT", 8080),
			IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		ImageStore: imageConfig{
			ServeRoot: env.Url("IMAGE_SERVE_ROOT", &url.URL{Scheme: "http", Host: "localhost:8080", Path: "/images/"}),
			Root:      env.String("IMAGE_ROOT", "./images"),
			MaxSize:   env.Int64("IMAGE_MAX_SIZE", 5*1024*1024),
			MaxWidth:  env.Int("IMAGE_MAX_WIDTH", 1920),
			MaxHeight: env.Int("IMAGE_MAX_HEIGHT", 1080),
		},
	}
}
