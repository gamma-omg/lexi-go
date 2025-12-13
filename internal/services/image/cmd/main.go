package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gamma-omg/lexi-go/internal/pkg/middleware"
	"github.com/gamma-omg/lexi-go/internal/pkg/router"
	"github.com/gamma-omg/lexi-go/internal/services/image/internal/config"
	"github.com/gamma-omg/lexi-go/internal/services/image/internal/rest"
	"github.com/gamma-omg/lexi-go/internal/services/image/internal/service"
)

func run(ctx context.Context) error {
	slog.Info("starting image service")

	cfg := config.FromEnv()
	srv := service.NewImageService(service.ImageServiceConfig{
		ServeRoot: cfg.ImageStore.ServeRoot,
		Root:      cfg.ImageStore.Root,
		MaxWidth:  cfg.ImageStore.MaxWidth,
		MaxHeight: cfg.ImageStore.MaxHeight,
	})

	r := router.New()
	r.Use(middleware.Recover(), middleware.Log())
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	auth := r.SubRouter("/api/v1/")
	auth.Use(middleware.Auth(cfg.AuthSecret))

	api := rest.NewAPI(srv, cfg.ImageStore.MaxSize, cfg.ImageStore.Root)
	r.Handle("/", api)

	httpSrv := &http.Server{
		Addr:         cfg.Http.ListenAddr,
		IdleTimeout:  cfg.Http.IdleTimeout,
		ReadTimeout:  cfg.Http.ReadTimeout,
		WriteTimeout: cfg.Http.WriteTimeout,
		Handler:      r,
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Http.ShutdownTimeout)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("image service exited with error", "error", err)
		os.Exit(1)
	}
}
