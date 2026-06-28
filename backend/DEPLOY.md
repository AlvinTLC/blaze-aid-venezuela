# Deploying BlazeAid Hub — Backend

The image is a static distroless binary that **self-migrates on boot** (app schema
+ River), so deploy = run the image against a Postgres and set env vars. The
worker is the same image with entrypoint `/worker`.

## 1. Database

Postgres 16 with the **TimescaleDB** and **pgvector** extensions (the app's
`001_init.sql` creates a hypertable and a `vector` column). Options:

- **Self-host**: `timescale/timescaledb-ha:pg16` (bundles both) — see `docker-compose.prod.yml`.
- **Managed**: Timescale Cloud (native), or any Postgres where you can
  `CREATE EXTENSION timescaledb, vector`. Plain Neon/Supabase do **not** ship
  TimescaleDB — use Timescale Cloud or self-host if you need the `events` hypertable.

## 2. Required environment

| Var | Required | Notes |
|-----|----------|-------|
| `ENV` | yes | set to `production` (enables the default-secret guard) |
| `DATABASE_URL` | yes | `postgres://user:pass@host:5432/db?sslmode=require` |
| `JWT_SECRET` | yes | strong random value; boot fails if left default in prod |
| `CORS_ORIGINS` | yes | the frontend origin(s), comma-separated |
| `APP_BASE_URL` | rec | absolute base for magic-login links, e.g. `https://api.blazeaid.app` |
| `SMTP_HOST/PORT/USER/PASS/SMTP_FROM/SMTP_TLS` | rec | real login email in prod; unset = links only logged |
| `RATE_LIMIT_RPM` | no | default 100/min (0 disables) |
| `MAX_BODY_BYTES` | no | default 1 MiB |
| `ADDR` | no | default `:8080` |

Generate a secret: `openssl rand -base64 48`.

## 3. The image

CI builds and pushes `harbor.blaze.do/blaze-aid-backend:{latest,sha}` on every
push to `main` **when the `HARBOR_USERNAME`/`HARBOR_PASSWORD` repo secrets are
set**. Without a registry you can build locally:

```sh
docker build -t blaze-aid-backend ./backend
```

## 4. Run

### Self-hosted / VPS (Docker Compose)
```sh
cd backend
cp .env.example .env   # fill in DATABASE_URL, JWT_SECRET, CORS_ORIGINS, SMTP_*
docker compose -f docker-compose.prod.yml up -d
curl https://your-host/healthz
```

### Fly.io
```sh
fly launch --image harbor.blaze.do/blaze-aid-backend:latest --no-deploy
fly secrets set ENV=production JWT_SECRET=... DATABASE_URL=... CORS_ORIGINS=... APP_BASE_URL=... SMTP_HOST=...
fly deploy
# worker as a second process/app with the same image, entrypoint /worker
```

### Railway / Render / EasyPanel
- New service from the image (or the repo + `backend/Dockerfile`).
- Add a Postgres (Timescale-capable) and set the env vars above.
- Add a second service for the worker: same image, start command `/worker`.
- Expose the api service on `:8080`; health check path `/healthz`.

## 5. Verify

```sh
curl https://HOST/healthz                 # 200 ok
curl https://HOST/openapi.json | head     # contract for the frontend
curl https://HOST/api/v1/stats            # dashboard data
```

Migrations apply automatically on first boot (tracked in `schema_migrations`).
Scale: run multiple `api` replicas freely (migrations are advisory-locked); run
one or more `worker` replicas to process the webhook queue.
