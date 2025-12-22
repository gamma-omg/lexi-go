package config_test

import (
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/image/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFromEnv(t *testing.T) {
	t.Setenv("HTTP_LISTEN_ADDR", ":9090")
	t.Setenv("HTTP_IDLE_TIMEOUT", "70s")
	t.Setenv("HTTP_READ_TIMEOUT", "40s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "50s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("AUTH_SECRET", "supersecret")
	t.Setenv("IMAGE_MAX_SIZE", "12345")
	t.Setenv("IMAGE_MAX_WIDTH", "2560")
	t.Setenv("IMAGE_MAX_HEIGHT", "1440")
	t.Setenv("IMAGE_ROOT", "./my_images")
	t.Setenv("IMAGE_SERVE_ROOT", "http://cdn.example.com/images/")

	cfg := config.FromEnv()

	assert.Equal(t, "supersecret", cfg.AuthSecret)
	assert.Equal(t, int64(12345), cfg.ImageStore.MaxSize)
	assert.Equal(t, 2560, cfg.ImageStore.MaxWidth)
	assert.Equal(t, 1440, cfg.ImageStore.MaxHeight)
	assert.Equal(t, "./my_images", cfg.ImageStore.Root)
	assert.Equal(t, ":9090", cfg.HTTP.ListenAddr)
	assert.Equal(t, 70*time.Second, cfg.HTTP.IdleTimeout)
	assert.Equal(t, 40*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 50*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, 15*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, "http://cdn.example.com/images/", cfg.ImageStore.ServeRoot.String())
}

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("AUTH_SECRET", "test")
	cfg := config.FromEnv()

	assert.Equal(t, "test", cfg.AuthSecret)
	assert.Equal(t, int64(5*1024*1024), cfg.ImageStore.MaxSize)
	assert.Equal(t, 1920, cfg.ImageStore.MaxWidth)
	assert.Equal(t, 1080, cfg.ImageStore.MaxHeight)
	assert.Equal(t, ":8080", cfg.HTTP.ListenAddr)
	assert.Equal(t, 60*time.Second, cfg.HTTP.IdleTimeout)
	assert.Equal(t, 30*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.HTTP.WriteTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTP.ShutdownTimeout)
	assert.Equal(t, "./images", cfg.ImageStore.Root)
	assert.Equal(t, "http://localhost:8080/images/", cfg.ImageStore.ServeRoot.String())
}
