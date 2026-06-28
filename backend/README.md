# BlazeAid Hub — Backend

Unified open-source platform aggregating post-earthquake tech-aid efforts in Venezuela.

## Stack

Go 1.23 · [Huma v2](https://huma.rocks) (chi adapter) · pgx/pgxpool · PostgreSQL (pgvector, TimescaleDB-ready) · Redis · River (async jobs) · Docker distroless.

## Layout

```
cmd/api               HTTP API entrypoint (enqueues webhook jobs)
cmd/worker            River worker entrypoint (processes webhook jobs)
internal/server       config + HTTP wiring
internal/domain/*     aidproject, resource, missing, volunteer, sync models
internal/handler      Huma operation handlers (ingest, sync, webhook, auth)
internal/repository   pgx data-access layer
internal/jobs         River job args, workers, migration + transactional enqueuer
internal/auth         session JWT issue/verify
migrations            SQL schema (auto-applied on first DB boot)
```

## Run (Docker)

```sh
docker compose up -d --build
curl localhost:8080/healthz
open http://localhost:8080/docs   # OpenAPI / Swagger UI from Huma
```

Postgres (TimescaleDB + pgvector) listens on `5432`, Redis on `6379`, the API on
`8080`. If any clash with something already running locally, override the host
ports without editing the file:

```sh
PG_PORT=5433 REDIS_PORT=6380 API_PORT=8090 docker compose up -d --build
```

## Run (local)

```sh
docker compose up -d postgres redis
cp .env.example .env
go run ./cmd/api
```

## P0 endpoints (`/api/v1`)

| Method | Path                  | Purpose                                  |
|--------|-----------------------|------------------------------------------|
| Method | Path                  | Auth   | Purpose                                |
|--------|-----------------------|--------|----------------------------------------|
| POST   | `/ingest/project`     | Bearer | Upsert an aid project                  |
| POST   | `/ingest/resource`    | Bearer | Upsert a resource                      |
| POST   | `/ingest/missing`     | Bearer | Upsert a missing-person report         |
| POST   | `/ingest/volunteer`   | Bearer | Upsert a volunteer                     |
| GET    | `/sync?since=&limit=` | public | Pull entity changes after a cursor     |
| POST   | `/webhook/{source}`   | public | Accept a raw inbound payload (queued)  |
| POST   | `/magic-login`        | public | Issue a passwordless login token       |
| POST   | `/auth/verify`        | public | Exchange a magic token for a session JWT |

All ingest endpoints are **idempotent**, keyed by `(source, external_id)`.
`/sync` uses an `updated_at` cursor; pass the returned `cursor` as the next `since`.

### Async webhook processing (River)

`POST /webhook/{source}` does not process inline. It writes the raw payload to
`webhooks_log` and enqueues a River job **in the same transaction** (exactly-once,
no orphaned work), returning `202 queued`. The `worker` service then routes the
payload to its typed table by the payload's `entity` field
(`project|resource|missing|volunteer`), records a row in the `events` hypertable,
and marks the webhook processed. Unknown/unparseable payloads are logged as
`unrouted` (not a failure); real DB errors are retried by River's default backoff.

```jsonc
// POST /api/v1/webhook/redcross
{ "entity": "missing", "external_id": "rc-42", "full_name": "Maria Lopez", "last_seen_region": "Vargas" }
```

`docker compose up` starts both `api` and `worker` against the same Postgres.

### Auth flow

1. `POST /api/v1/magic-login {email}` → mints a single-use magic token (returned
   in the body in dev; emailed in production).
2. `POST /api/v1/auth/verify {token}` → burns the magic token and returns a signed
   HS256 **session JWT** (`access_token`, 24h TTL).
3. Call protected endpoints with `Authorization: Bearer <access_token>`.

## Data model

Typed entity tables (`aid_projects`, `resources`, `missing_persons`, `volunteers`)
keyed by `(source, external_id)` for idempotent upserts. `webhooks_log` stores raw
inbound payloads. `events` is a **TimescaleDB hypertable** (partitioned by
`occurred_at`) reserved for the append-only ingestion/audit stream.

## Security (P0 status — read before deploying)

This is a beta skeleton. Known limitations, tracked for hardening:

- **Ingest endpoints require a Bearer JWT** (HS256, signed with `JWT_SECRET`).
  `/sync` and `/webhook/{source}` remain public; webhook source authentication
  (signatures per provider) is a separate task.
- **`magic-login` is a stub delivery.** It returns the token in the response body
  **only in non-production**; with `ENV=production` the token is suppressed and
  must be delivered out-of-band (email). The token is consumed by `/auth/verify`.
- **No default secrets in prod.** The app refuses to boot when `ENV=production`
  and `JWT_SECRET` is the development default (`config.Validate`). Always set a
  strong `JWT_SECRET` and real DB credentials via the environment.
- **Webhook payloads are stored verbatim** as `jsonb`; validate/sanitize per
  source when wiring real processing.

## Notes / TODO (beyond P0)

- Per-provider webhook signature verification (HMAC) on `/webhook/{source}`.
- `GET /admin/jobs` for River job monitoring (retryable/discarded).
- Add embedding generation (pgvector column already provisioned).
- Use the `events` hypertable for ingestion metrics / time-series queries.
