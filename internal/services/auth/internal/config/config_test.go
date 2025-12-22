package config_test

import (
	"testing"
	"time"

	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/config"
	"github.com/stretchr/testify/require"
)

func TestFromEnv(t *testing.T) {
	t.Setenv("HTTP_LISTEN_ADDR", ":9090")
	t.Setenv("HTTP_READ_TIMEOUT", "45s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "45s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "90s")
	t.Setenv("HTTP_SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("JWT_ACCESS_SECRET", "accesssecret")
	t.Setenv("JWT_REFRESH_SECRET", "refreshsecret")
	t.Setenv("JWT_ACCESS_TTL", "20m")
	t.Setenv("JWT_REFRESH_TTL", "14h")

	cfg := config.FromEnv()

	require.Equal(t, ":9090", cfg.HTTP.ListenAddr)
	require.Equal(t, 45*time.Second, cfg.HTTP.ReadTimeout)
	require.Equal(t, 45*time.Second, cfg.HTTP.WriteTimeout)
	require.Equal(t, 90*time.Second, cfg.HTTP.IdleTimeout)
	require.Equal(t, 15*time.Second, cfg.HTTP.ShutdownTimeout)
	require.Equal(t, "accesssecret", cfg.JWT.AccessSecret)
	require.Equal(t, "refreshsecret", cfg.JWT.RefreshSecret)
	require.Equal(t, 20*time.Minute, cfg.JWT.AccessTTL)
	require.Equal(t, 14*time.Hour, cfg.JWT.RefreshTTL)
}

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("JWT_ACCESS_SECRET", "defaultaccess")
	t.Setenv("JWT_REFRESH_SECRET", "defaultrefresh")
	cfg := config.FromEnv()

	require.Equal(t, ":8080", cfg.HTTP.ListenAddr)
	require.Equal(t, 30*time.Second, cfg.HTTP.ReadTimeout)
	require.Equal(t, 30*time.Second, cfg.HTTP.WriteTimeout)
	require.Equal(t, 60*time.Second, cfg.HTTP.IdleTimeout)
	require.Equal(t, 10*time.Second, cfg.HTTP.ShutdownTimeout)
	require.Equal(t, "defaultaccess", cfg.JWT.AccessSecret)
	require.Equal(t, "defaultrefresh", cfg.JWT.RefreshSecret)
	require.Equal(t, 15*time.Minute, cfg.JWT.AccessTTL)
	require.Equal(t, 7*24*time.Hour, cfg.JWT.RefreshTTL)
}
