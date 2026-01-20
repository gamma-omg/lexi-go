package config

import (
	"time"

	"github.com/gamma-omg/lexi-go/internal/pkg/env"
)

// Config holds the entire configuration for the auth service
type Config struct {
	HTTP     httpConfig
	JWT      jwtConfig
	DB       dbConfig
	OAuth    oauthConfig
	RedisOTC otcRedisConfig
}

type httpConfig struct {
	ListenAddr      string
	ListenPort      int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type jwtConfig struct {
	AccessSecret     string
	RefreshSecret    string
	Issuer           string
	AlgorithmAccess  string
	AlgorithmRefresh string
	AccessTTL        time.Duration
	RefreshTTL       time.Duration
}

type dbConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

type googleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type oauthConfig struct {
	Google googleConfig
}

type otcRedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	CodeTTL  time.Duration
}

// FromEnv loads the configuration from environment variables
func FromEnv() Config {
	return Config{
		HTTP: httpConfig{
			ListenAddr:      env.String("HTTP_LISTEN_ADDR", ""),
			ListenPort:      env.Int("HTTP_LISTEN_PORT", 8080),
			ReadTimeout:     env.Duration("HTTP_READ_TIMEOUT", 30*time.Second),
			WriteTimeout:    env.Duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     env.Duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: env.Duration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		JWT: jwtConfig{
			AccessSecret:     env.RequireString("JWT_ACCESS_SECRET"),
			RefreshSecret:    env.RequireString("JWT_REFRESH_SECRET"),
			Issuer:           env.String("JWT_ISSUER", "lexigo-auth-service"),
			AlgorithmAccess:  env.String("JWT_ALGORITHM_ACCESS", "ES256"),
			AlgorithmRefresh: env.String("JWT_ALGORITHM_REFRESH", "HS256"),
			AccessTTL:        env.Duration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:       env.Duration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		DB: dbConfig{
			Host:     env.String("DB_HOST", "localhost"),
			Port:     env.String("DB_PORT", "5432"),
			User:     env.String("DB_USER", "postgres"),
			Password: env.String("DB_PASSWORD", "password"),
			Name:     env.String("DB_NAME", "auth_service"),
		},
		OAuth: oauthConfig{
			Google: googleConfig{
				ClientID:     env.RequireString("OAUTH_GOOGLE_CLIENT_ID"),
				ClientSecret: env.RequireString("OAUTH_GOOGLE_CLIENT_SECRET"),
				RedirectURL:  env.String("OAUTH_GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
			},
		},
		RedisOTC: otcRedisConfig{
			Host:     env.String("OTC_REDIS_HOST", "localhost"),
			Port:     env.String("OTC_REDIS_PORT", "6379"),
			Password: env.String("OTC_REDIS_PASSWORD", ""),
			DB:       env.Int("OTC_REDIS_DB", 0),
			CodeTTL:  env.Duration("OTC_CODE_TTL", 10*time.Second),
		},
	}
}
