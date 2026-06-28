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

## Notes / TODO (beyond P0)

- Wire River for async processing of `raw_events` rows.
- `magic-login` currently returns the token directly (beta); production must email it.
- Add embedding generation (pgvector column already provisioned).
- Promote to TimescaleDB hypertables for time-series-heavy entities.
