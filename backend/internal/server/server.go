package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/handler"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// Run wires the dependency graph and serves until ctx is cancelled.
func Run(ctx context.Context, logger *slog.Logger) error {
	cfg := LoadConfig()
	if err := cfg.Validate(); err != nil {
		return err
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	repo := repository.New(pool)

	router := chi.NewMux()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))

	// Liveness/readiness probe (outside the OpenAPI surface).
	router.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "db unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	humaConfig := huma.DefaultConfig("BlazeAid Hub API", "0.1.0-beta1")
	humaConfig.Info.Description = "Unified open-source platform for post-earthquake tech aid in Venezuela."
	api := humachi.New(router, humaConfig)

	h := handler.New(repo, cfg.JWTSecret, cfg.IsProduction(), logger)
	h.Register(api)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("BlazeAid Hub API listening", "addr", cfg.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	logger.Info("BlazeAid Hub API stopped")
	return nil
}
