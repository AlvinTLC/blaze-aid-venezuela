package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/jobs"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/migrate"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/server"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/migrations"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, logger); err != nil {
		logger.Error("worker exited with error", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger) error {
	cfg := server.LoadConfig()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, migrations.FS); err != nil {
		return err
	}
	if err := jobs.Migrate(ctx, pool); err != nil {
		return err
	}

	client, err := jobs.NewWorkerClient(pool, repository.New(pool), logger)
	if err != nil {
		return err
	}

	if err := client.Start(ctx); err != nil {
		return err
	}
	logger.Info("BlazeAid Hub worker started")

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := client.Stop(shutdownCtx); err != nil {
		return err
	}
	logger.Info("BlazeAid Hub worker stopped")
	return nil
}
