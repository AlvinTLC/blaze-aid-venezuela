package jobs_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/jobs"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/testdb"
)

var pool *pgxpool.Pool

func TestMain(m *testing.M) {
	ctx := context.Background()
	p, cleanup, err := testdb.New(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "testdb setup failed:", err)
		os.Exit(1)
	}
	pool = p
	if err := jobs.Migrate(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "river migrate failed:", err)
		cleanup()
		os.Exit(1)
	}
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func reset(t *testing.T) {
	t.Helper()
	if err := testdb.Truncate(context.Background(), pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// The worker routes a payload with entity=project to aid_projects, records an
// event, and marks the webhook processed.
func TestWebhookProcess_RoutesToTypedTable(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	id, err := repo.InsertWebhookLog(ctx, "form",
		[]byte(`{"entity":"project","external_id":"wh-1","title":"Mesh Caracas","region":"DC"}`))
	if err != nil {
		t.Fatal(err)
	}

	worker := &jobs.WebhookProcessWorker{Repo: repo}
	if err := worker.Work(ctx, &river.Job[jobs.WebhookProcessArgs]{
		Args: jobs.WebhookProcessArgs{WebhookID: id, Source: "form"},
	}); err != nil {
		t.Fatalf("work: %v", err)
	}

	var projects, events int
	mustScan(t, `SELECT count(*) FROM aid_projects WHERE external_id='wh-1'`, &projects)
	mustScan(t, `SELECT count(*) FROM events WHERE entity='project'`, &events)
	if projects != 1 {
		t.Fatalf("expected project row created, got %d", projects)
	}
	if events != 1 {
		t.Fatalf("expected one event recorded, got %d", events)
	}

	var processed bool
	mustScan(t, fmt.Sprintf(`SELECT processed FROM webhooks_log WHERE id='%s'`, id), &processed)
	if !processed {
		t.Fatal("expected webhook marked processed")
	}
}

// An unknown entity is a non-error: the webhook is processed and logged as
// "unrouted" with no typed row created.
func TestWebhookProcess_UnknownEntityIsUnrouted(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	id, err := repo.InsertWebhookLog(ctx, "mystery", []byte(`{"entity":"alien","foo":1}`))
	if err != nil {
		t.Fatal(err)
	}

	worker := &jobs.WebhookProcessWorker{Repo: repo}
	if err := worker.Work(ctx, &river.Job[jobs.WebhookProcessArgs]{
		Args: jobs.WebhookProcessArgs{WebhookID: id, Source: "mystery"},
	}); err != nil {
		t.Fatalf("work: %v", err)
	}

	var unrouted int
	mustScan(t, `SELECT count(*) FROM events WHERE kind='unrouted'`, &unrouted)
	if unrouted != 1 {
		t.Fatalf("expected one unrouted event, got %d", unrouted)
	}
}

// Full pipeline: enqueue (webhooks_log + river_job in one tx) -> a running River
// client processes the job -> the typed row lands and the webhook is processed.
func TestEnqueueAndProcess_EndToEnd(t *testing.T) {
	reset(t)
	ctx := context.Background()
	repo := repository.New(pool)

	enq, err := jobs.NewEnqueuer(pool)
	if err != nil {
		t.Fatalf("enqueuer: %v", err)
	}

	whID, err := enq.EnqueueWebhook(ctx, "form",
		[]byte(`{"entity":"resource","external_id":"wh-e2e","name":"Agua","quantity":100,"unit":"L"}`))
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	var jobCount int
	mustScan(t, `SELECT count(*) FROM river_job`, &jobCount)
	if jobCount != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", jobCount)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client, err := jobs.NewWorkerClient(pool, repo, logger)
	if err != nil {
		t.Fatalf("worker client: %v", err)
	}
	if err := client.Start(ctx); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = client.Stop(stopCtx)
	}()

	// Wait for the worker to process the job.
	deadline := time.Now().Add(20 * time.Second)
	for {
		var processed bool
		mustScan(t, fmt.Sprintf(`SELECT processed FROM webhooks_log WHERE id='%s'`, whID), &processed)
		if processed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the worker to process the webhook")
		}
		time.Sleep(150 * time.Millisecond)
	}

	var resources int
	mustScan(t, `SELECT count(*) FROM resources WHERE external_id='wh-e2e'`, &resources)
	if resources != 1 {
		t.Fatalf("expected resource row from worker, got %d", resources)
	}
}

func mustScan(t *testing.T, query string, dest ...any) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), query).Scan(dest...); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
}
