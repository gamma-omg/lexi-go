package config_test

import (
	"testing"

	"github.com/gamma-omg/lexi-go/internal/services/words/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFromEnv(t *testing.T) {
	t.Setenv("LISTEN_ADDR", ":9090")
	t.Setenv("AUTH_SECRET", "supersecret")
	t.Setenv("TAGS_CACHE_KEYS", "200")
	t.Setenv("TAGS_CACHE_COST", "300")
	t.Setenv("DB_HOST", "db.example.com")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_USER", "testuser")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "testdb")

	cfg := config.FromEnv()

	assert.Equal(t, ":9090", cfg.ListenAddr)
	assert.Equal(t, "supersecret", cfg.AuthSecret)
	assert.Equal(t, int64(200), cfg.TagsMaxKeys)
	assert.Equal(t, int64(300), cfg.TagsMaxCost)
	assert.Equal(t, "db.example.com", cfg.DB.Host)
	assert.Equal(t, "6543", cfg.DB.Port)
	assert.Equal(t, "testuser", cfg.DB.User)
	assert.Equal(t, "testpass", cfg.DB.Password)
	assert.Equal(t, "testdb", cfg.DB.Name)
}

func TestFromEnv_Defaults(t *testing.T) {
	cfg := config.FromEnv()

	assert.Equal(t, ":8080", cfg.ListenAddr)
	assert.Equal(t, "", cfg.AuthSecret)
	assert.Equal(t, int64(10000), cfg.TagsMaxKeys)
	assert.Equal(t, int64(10000), cfg.TagsMaxCost)
	assert.Equal(t, "localhost", cfg.DB.Host)
	assert.Equal(t, "5432", cfg.DB.Port)
	assert.Equal(t, "postgres", cfg.DB.User)
	assert.Equal(t, "password", cfg.DB.Password)
	assert.Equal(t, "words_service", cfg.DB.Name)
}
