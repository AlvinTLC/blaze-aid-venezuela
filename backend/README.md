# BlazeAid Hub — Backend

Unified open-source platform aggregating post-earthquake tech-aid efforts in Venezuela.

## Stack

Go 1.23 · [Huma v2](https://huma.rocks) (chi adapter) · pgx/pgxpool · PostgreSQL (pgvector, TimescaleDB-ready) · Redis · River (async jobs) · Docker distroless.

## Layout

```
cmd/api               entrypoint
internal/server       config + HTTP wiring
internal/domain/*     aidproject, resource, missing, volunteer, sync models
internal/handler      Huma operation handlers (ingest, sync, webhook, auth)
internal/repository   pgx data-access layer
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
| POST   | `/ingest/project`     | Upsert an aid project                    |
| POST   | `/ingest/resource`    | Upsert a resource                        |
| POST   | `/ingest/missing`     | Upsert a missing-person report           |
| POST   | `/ingest/volunteer`   | Upsert a volunteer                       |
| GET    | `/sync?since=&limit=` | Pull entity changes after a cursor       |
| POST   | `/webhook/{source}`   | Accept a raw inbound payload (queued)    |
| POST   | `/magic-login`        | Issue a passwordless login token         |

All ingest endpoints are **idempotent**, keyed by `(source, external_id)`.
`/sync` uses an `updated_at` cursor; pass the returned `cursor` as the next `since`.

## Data model

Typed entity tables (`aid_projects`, `resources`, `missing_persons`, `volunteers`)
keyed by `(source, external_id)` for idempotent upserts. `webhooks_log` stores raw
inbound payloads. `events` is a **TimescaleDB hypertable** (partitioned by
`occurred_at`) reserved for the append-only ingestion/audit stream.

## Security (P0 status — read before deploying)

This is a beta skeleton. Known limitations, tracked for hardening:

- **Ingest endpoints are unauthenticated.** P0 accepts open ingestion; add an API
  key / signed-source check before exposing publicly.
- **`magic-login` is a stub.** It returns the token in the response body **only in
  non-production**. With `ENV=production` the token is suppressed and must be
  delivered out-of-band (email); a `/auth/verify` consumer is still TODO.
- **No default secrets in prod.** The app refuses to boot when `ENV=production`
  and `JWT_SECRET` is the development default (`config.Validate`). Always set a
  strong `JWT_SECRET` and real DB credentials via the environment.
- **Webhook payloads are stored verbatim** as `jsonb`; validate/sanitize per
  source when wiring real processing.

## Notes / TODO (beyond P0)

- Wire River for async processing of `webhooks_log` rows.
- Implement `/auth/verify` to consume magic tokens and issue a session JWT.
- Add embedding generation (pgvector column already provisioned).
- Use the `events` hypertable for ingestion metrics / time-series queries.
