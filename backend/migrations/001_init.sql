-- 001_init.sql — BlazeAid Hub initial schema (P0)
-- Stack: Postgres + TimescaleDB + pgvector. Idempotent.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;
-- TimescaleDB is optional for P0; uncomment when the image provides it.
-- CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Tech / relief projects aggregated into the hub.
CREATE TABLE IF NOT EXISTS aid_projects (
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source      text NOT NULL,
    external_id text NOT NULL,
    title       text NOT NULL,
    description text NOT NULL DEFAULT '',
    category    text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'active',
    region      text NOT NULL DEFAULT '',
    lat         double precision,
    lng         double precision,
    contact     text NOT NULL DEFAULT '',
    url         text NOT NULL DEFAULT '',
    embedding   vector(384),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id)
);

-- Physical resources / supplies (water, fuel, tools, connectivity...).
CREATE TABLE IF NOT EXISTS resources (
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source      text NOT NULL,
    external_id text NOT NULL,
    type        text NOT NULL DEFAULT '',
    name        text NOT NULL,
    quantity    double precision NOT NULL DEFAULT 0,
    unit        text NOT NULL DEFAULT '',
    status      text NOT NULL DEFAULT 'available',
    region      text NOT NULL DEFAULT '',
    lat         double precision,
    lng         double precision,
    contact     text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id)
);

-- Missing persons reports.
CREATE TABLE IF NOT EXISTS missing_persons (
    id                uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source            text NOT NULL,
    external_id       text NOT NULL,
    full_name         text NOT NULL,
    age               int,
    description       text NOT NULL DEFAULT '',
    last_seen_region  text NOT NULL DEFAULT '',
    last_seen_at      timestamptz,
    status            text NOT NULL DEFAULT 'missing',
    contact           text NOT NULL DEFAULT '',
    photo_url         text NOT NULL DEFAULT '',
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id)
);

-- Volunteers offering skills / time.
CREATE TABLE IF NOT EXISTS volunteers (
    id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source       text NOT NULL,
    external_id  text NOT NULL,
    full_name    text NOT NULL,
    skills       text[] NOT NULL DEFAULT '{}',
    availability text NOT NULL DEFAULT '',
    region       text NOT NULL DEFAULT '',
    contact      text NOT NULL DEFAULT '',
    status       text NOT NULL DEFAULT 'available',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, external_id)
);

-- Raw inbound webhook payloads, processed asynchronously (River) later.
CREATE TABLE IF NOT EXISTS raw_events (
    id          uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    source      text NOT NULL,
    payload     jsonb NOT NULL,
    processed   boolean NOT NULL DEFAULT false,
    received_at timestamptz NOT NULL DEFAULT now()
);

-- Passwordless magic-login tokens.
CREATE TABLE IF NOT EXISTS magic_tokens (
    token      text PRIMARY KEY,
    email      text NOT NULL,
    expires_at timestamptz NOT NULL,
    used       boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Sync cursors ride on updated_at; index every entity for fast "changed since".
CREATE INDEX IF NOT EXISTS idx_aid_projects_updated_at    ON aid_projects (updated_at);
CREATE INDEX IF NOT EXISTS idx_resources_updated_at       ON resources (updated_at);
CREATE INDEX IF NOT EXISTS idx_missing_persons_updated_at ON missing_persons (updated_at);
CREATE INDEX IF NOT EXISTS idx_volunteers_updated_at      ON volunteers (updated_at);
CREATE INDEX IF NOT EXISTS idx_raw_events_processed       ON raw_events (processed, received_at);
