// Package jobs wires River (Postgres-backed job queue) for async webhook
// processing: schema migration, the worker client, and a transactional enqueuer.
package jobs

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// riverMigrateLockKey serializes River migrations across concurrent api/worker
// boots (River's migrator has no internal locking). Distinct from the app key.
const riverMigrateLockKey int64 = 947563022

// Migrate creates/updates River's own tables (river_job, river_leader, ...).
// Idempotent and safe to call concurrently from api and worker: a Postgres
// advisory lock ensures only one instance migrates at a time.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", riverMigrateLockKey); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", riverMigrateLockKey)
	}()

	migrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return err
	}
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	return err
}

// NewWorkerClient builds a River client that runs the BlazeAid workers. Call
// Start(ctx) to begin consuming and Stop(ctx) for graceful shutdown.
func NewWorkerClient(pool *pgxpool.Pool, repo *repository.Repository, logger *slog.Logger) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &WebhookProcessWorker{Repo: repo})

	return river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger: logger,
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
}

// Enqueuer persists a webhook payload and enqueues its processing job atomically.
type Enqueuer struct {
	pool   *pgxpool.Pool
	client *river.Client[pgx.Tx]
}

// NewEnqueuer builds an insert-only River client (no workers) for the API process.
func NewEnqueuer(pool *pgxpool.Pool) (*Enqueuer, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{})
	if err != nil {
		return nil, err
	}
	return &Enqueuer{pool: pool, client: client}, nil
}

// EnqueueWebhook inserts the raw payload into webhooks_log and enqueues a
// WebhookProcess job in the SAME transaction, so the job and its data commit (or
// roll back) together — no lost or orphaned work.
func (e *Enqueuer) EnqueueWebhook(ctx context.Context, source string, payload []byte) (string, error) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO webhooks_log (source, payload) VALUES ($1, $2) RETURNING id`,
		source, payload).Scan(&id); err != nil {
		return "", err
	}

	if _, err := e.client.InsertTx(ctx, tx, WebhookProcessArgs{WebhookID: id, Source: source}, nil); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id, nil
}
