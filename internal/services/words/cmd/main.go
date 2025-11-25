package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/config"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/rest"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/service"
	"github.com/gamma-omg/lexi-go/internal/services/words/internal/store"
)

func run(ctx context.Context) error {
	slog.Info("starting words service")

	cfg := config.FromEnv()
	store, err := store.NewPostgresStore(store.PostgresConfig{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DB:       cfg.DB.Name,
	})
	if err != nil {
		return err
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r := router.New()
	r.Use(middleware.Auth(cfg.AuthSecret))

	srv := service.NewWordsService(store)
	api := rest.NewAPI(srv)
	api.Register(r)

	http.Handle("/api/v1/", r)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server", "addr", cfg.ListenAddr)
		errCh <- http.ListenAndServe(cfg.ListenAddr, nil)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func main() {
	if err := run(context.Background()); err != nil {
		slog.Error("service exited with error", "error", err)
		os.Exit(1)
	}
}
