package config_test

import (
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/image/internal/config"
	"github.com/magiconair/properties/assert"
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

	assert.Equal(t, cfg.AuthSecret, "supersecret")
	assert.Equal(t, cfg.ImageStore.MaxSize, int64(12345))
	assert.Equal(t, cfg.ImageStore.MaxWidth, 2560)
	assert.Equal(t, cfg.ImageStore.MaxHeight, 1440)
	assert.Equal(t, cfg.ImageStore.Root, "./my_images")
	assert.Equal(t, cfg.Http.ListenAddr, ":9090")
	assert.Equal(t, cfg.Http.IdleTimeout, 70*time.Second)
	assert.Equal(t, cfg.Http.ReadTimeout, 40*time.Second)
	assert.Equal(t, cfg.Http.WriteTimeout, 50*time.Second)
	assert.Equal(t, cfg.Http.ShutdownTimeout, 15*time.Second)
	assert.Equal(t, cfg.ImageStore.ServeRoot.String(), "http://cdn.example.com/images/")
}

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("AUTH_SECRET", "test")
	cfg := config.FromEnv()

	assert.Equal(t, cfg.AuthSecret, "test")
	assert.Equal(t, cfg.ImageStore.MaxSize, int64(5*1024*1024))
	assert.Equal(t, cfg.ImageStore.MaxWidth, 1920)
	assert.Equal(t, cfg.ImageStore.MaxHeight, 1080)
	assert.Equal(t, cfg.Http.ListenAddr, ":8080")
	assert.Equal(t, cfg.Http.IdleTimeout, 60*time.Second)
	assert.Equal(t, cfg.Http.ReadTimeout, 30*time.Second)
	assert.Equal(t, cfg.Http.WriteTimeout, 30*time.Second)
	assert.Equal(t, cfg.Http.ShutdownTimeout, 10*time.Second)
	assert.Equal(t, cfg.ImageStore.Root, "./images")
	assert.Equal(t, cfg.ImageStore.ServeRoot.String(), "http://localhost:8080/images/")
}
