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
	t.Setenv("JWT_ACCESS_SECRET", "access_secret")
	t.Setenv("JWT_REFRESH_SECRET", "refresh_secret")
	t.Setenv("JWT_ACCESS_TTL", "20m")
	t.Setenv("JWT_REFRESH_TTL", "14h")
	t.Setenv("DB_HOST", "dbhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "dbuser")
	t.Setenv("DB_PASSWORD", "dbpassword")
	t.Setenv("DB_NAME", "auth_service")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "google_client_id")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "google_client_secret")
	t.Setenv("OAUTH_GOOGLE_REDIRECT_URL", "http://localhost:9090/auth/google/callback")

	cfg := config.FromEnv()

	require.Equal(t, ":9090", cfg.HTTP.ListenAddr)
	require.Equal(t, 45*time.Second, cfg.HTTP.ReadTimeout)
	require.Equal(t, 45*time.Second, cfg.HTTP.WriteTimeout)
	require.Equal(t, 90*time.Second, cfg.HTTP.IdleTimeout)
	require.Equal(t, 15*time.Second, cfg.HTTP.ShutdownTimeout)
	require.Equal(t, "access_secret", cfg.JWT.AccessSecret)
	require.Equal(t, "refresh_secret", cfg.JWT.RefreshSecret)
	require.Equal(t, 20*time.Minute, cfg.JWT.AccessTTL)
	require.Equal(t, 14*time.Hour, cfg.JWT.RefreshTTL)
	require.Equal(t, "dbhost", cfg.DB.Host)
	require.Equal(t, "5432", cfg.DB.Port)
	require.Equal(t, "dbuser", cfg.DB.User)
	require.Equal(t, "dbpassword", cfg.DB.Password)
	require.Equal(t, "auth_service", cfg.DB.Name)
	require.Equal(t, "google_client_id", cfg.OAuth.Google.ClientID)
	require.Equal(t, "google_client_secret", cfg.OAuth.Google.ClientSecret)
	require.Equal(t, "http://localhost:9090/auth/google/callback", cfg.OAuth.Google.RedirectURL)
}

func TestFromEnv_Defaults(t *testing.T) {
	t.Setenv("JWT_ACCESS_SECRET", "default_access")
	t.Setenv("JWT_REFRESH_SECRET", "default_refresh")
	t.Setenv("OAUTH_GOOGLE_CLIENT_ID", "client_id")
	t.Setenv("OAUTH_GOOGLE_CLIENT_SECRET", "secret")
	cfg := config.FromEnv()

	require.Equal(t, ":8080", cfg.HTTP.ListenAddr)
	require.Equal(t, 30*time.Second, cfg.HTTP.ReadTimeout)
	require.Equal(t, 30*time.Second, cfg.HTTP.WriteTimeout)
	require.Equal(t, 60*time.Second, cfg.HTTP.IdleTimeout)
	require.Equal(t, 10*time.Second, cfg.HTTP.ShutdownTimeout)
	require.Equal(t, "default_access", cfg.JWT.AccessSecret)
	require.Equal(t, "default_refresh", cfg.JWT.RefreshSecret)
	require.Equal(t, 15*time.Minute, cfg.JWT.AccessTTL)
	require.Equal(t, 7*24*time.Hour, cfg.JWT.RefreshTTL)
	require.Equal(t, "localhost", cfg.DB.Host)
	require.Equal(t, "5432", cfg.DB.Port)
	require.Equal(t, "postgres", cfg.DB.User)
	require.Equal(t, "password", cfg.DB.Password)
	require.Equal(t, "auth_service", cfg.DB.Name)
	require.Equal(t, "client_id", cfg.OAuth.Google.ClientID)
	require.Equal(t, "secret", cfg.OAuth.Google.ClientSecret)
	require.Equal(t, "http://localhost:8080/auth/google/callback", cfg.OAuth.Google.RedirectURL)
}
