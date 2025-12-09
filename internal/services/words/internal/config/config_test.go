package config_test

import (
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFromEnv(t *testing.T) {
	t.Setenv("HTTP_LISTEN_ADDR", ":9090")
	t.Setenv("HTTP_IDLE_TIMEOUT", "70s")
	t.Setenv("HTTP_READ_TIMEOUT", "40s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "50s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("AUTH_SECRET", "supersecret")
	t.Setenv("TAGS_CACHE_KEYS", "200")
	t.Setenv("TAGS_CACHE_COST", "300")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")
	t.Setenv("IMAGE_ENDPOINT", "http://example.com:8888/upload")
	t.Setenv("IMAGE_FIELD_NAME", "img")
	t.Setenv("IMAGE_FILE_NAME", "img.jpg")

	cfg := config.FromEnv()

	assert.Equal(t, "supersecret", cfg.AuthSecret)
	assert.Equal(t, int64(200), cfg.TagsMaxKeys)
	assert.Equal(t, int64(300), cfg.TagsMaxCost)
	assert.Equal(t, "db.example.com", cfg.DB.Host)
	assert.Equal(t, "6543", cfg.DB.Port)
	assert.Equal(t, "testuser", cfg.DB.User)
	assert.Equal(t, "testpass", cfg.DB.Password)
	assert.Equal(t, "testdb", cfg.DB.Name)
	assert.Equal(t, ":9090", cfg.Http.ListenAddr)
	assert.Equal(t, 70*time.Second, cfg.Http.IdleTimeout)
	assert.Equal(t, 40*time.Second, cfg.Http.ReadTimeout)
	assert.Equal(t, 50*time.Second, cfg.Http.WriteTimeout)
	assert.Equal(t, 15*time.Second, cfg.Http.ShutdownTimeout)
	assert.Equal(t, "http://example.com:8888/upload", cfg.Image.Endpoint)
	assert.Equal(t, "img", cfg.Image.FieldName)
	assert.Equal(t, "img.jpg", cfg.Image.FileName)
}

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("AUTH_SECRET", "test")
	cfg := config.FromEnv()

	assert.Equal(t, "test", cfg.AuthSecret)
	assert.Equal(t, int64(10000), cfg.TagsMaxKeys)
	assert.Equal(t, int64(10000), cfg.TagsMaxCost)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, "5432", cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.User)
	assert.Equal(t, "password", cfg.DB.Password)
	assert.Equal(t, "words_service", cfg.DB.Name)
	assert.Equal(t, ":8080", cfg.Http.ListenAddr)
	assert.Equal(t, 60*time.Second, cfg.Http.IdleTimeout)
	assert.Equal(t, 30*time.Second, cfg.Http.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.Http.WriteTimeout)
	assert.Equal(t, 10*time.Second, cfg.Http.ShutdownTimeout)
	assert.Equal(t, "http://localhost:9999/", cfg.Image.Endpoint)
	assert.Equal(t, "image", cfg.Image.FieldName)
	assert.Equal(t, "image.jpg", cfg.Image.FileName)
}
