package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/config"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/image"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/rest"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
)

func run(ctx context.Context) error {
	slog.Info("starting words service")

	cfg := config.FromEnv()
	db, err := store.NewPostgresDB(store.PostgresConfig{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DB:       cfg.DB.Name,
	})
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}

	store := store.NewPostgresStore(db)
	imgStore := image.NewRemoteStore(
		cfg.Image.Endpoint,
		cfg.Image.FieldName,
		cfg.Image.FileName,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := router.New()
	r.Use(middleware.Auth(cfg.AuthSecret))

	srv := service.NewWordsService(store, service.WordsServiceConfig{
		TagsCacheSize: cfg.TagsMaxKeys,
		TagsMaxCost:   cfg.TagsMaxCost,
	})
	api := rest.NewAPI(srv, imgStore)
	api.Register(r)

	mux.Handle("/api/v1/", r)
	server := &http.Server{
		Addr:         cfg.Http.ListenAddr,
		IdleTimeout:  cfg.Http.IdleTimeout,
		ReadTimeout:  cfg.Http.ReadTimeout,
		WriteTimeout: cfg.Http.WriteTimeout,
		Handler:      mux,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server", "addr", cfg.Http.ListenAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down HTTP server")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Http.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}

	slog.Info("words service stopped")
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.Error("words service exited with error", "error", err)
		os.Exit(1)
	}
}
