package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/config"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/oauth"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/provider"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/rest"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/service"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/store"
	"github.com/gamma-omg/lexi-go/internal/services/auth/internal/token"
)

func run(ctx context.Context) error {
	slog.Info("starting auth service")

	cfg := config.FromEnv()
	db, err := store.NewPostgresDB(store.PostgresConfig{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DB:       cfg.DB.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	pgs := store.NewPostgresStore(db)

	auth := oauth.NewAuthenticator()
	if err := registerProviders(ctx, auth, cfg); err != nil {
		return fmt.Errorf("failed to register oauth providers: %w", err)
	}

	srv := service.NewAuth(
		service.WithAuthenticator(auth),
		service.WithStore(pgs),
		service.WithAccessToken(token.NewJWTIssuer(token.JwtConfig{
			Secret: token.NewSecretString(cfg.JWT.AccessSecret),
			TTL:    cfg.JWT.AccessTTL,
		})),
		service.WithRefreshToken(token.NewJWTIssuer(token.JwtConfig{
			Secret: token.NewSecretString(cfg.JWT.RefreshSecret),
			TTL:    cfg.JWT.RefreshTTL,
		})),
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	api := rest.NewAPI(srv)
	mux.Handle("/api/v1/", api)

	httpSrv := &http.Server{
		Addr:         cfg.HTTP.ListenAddr,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		Handler:      mux,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server starting", "addr", httpSrv.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down HTTP server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func registerProviders(ctx context.Context, auth *oauth.Authenticator, cfg config.Config) error {
	prvGoogle, err := provider.NewGoogle(ctx, provider.GoogleConfig{
		ClientID:     cfg.OAuth.Google.ClientID,
		ClientSecret: cfg.OAuth.Google.ClientSecret,
		RedirectURL:  cfg.OAuth.Google.RedirectURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create google oauth provider: %w", err)
	}

	auth.Use("google", prvGoogle)
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("auth service terminated with error", "error", err)
		os.Exit(1)
	}
}
