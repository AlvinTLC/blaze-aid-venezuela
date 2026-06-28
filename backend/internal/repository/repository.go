package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	syncdom "github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/sync"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
)

// ErrInvalidToken is returned when a magic token is unknown, already used, or expired.
var ErrInvalidToken = errors.New("invalid, used, or expired magic token")

// Repository is a thin data-access layer over a pgx connection pool.
type Repository struct {
	pool *pgxpool.Pool
}

// New constructs a Repository backed by the given pool.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

// UpsertProject inserts or updates a project keyed by (source, external_id).
func (r *Repository) UpsertProject(ctx context.Context, p aidproject.AidProject) (string, error) {
	const q = `
INSERT INTO aid_projects (source, external_id, title, description, category, status, region, lat, lng, contact, url)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (source, external_id) DO UPDATE SET
  title=EXCLUDED.title, description=EXCLUDED.description, category=EXCLUDED.category,
  status=EXCLUDED.status, region=EXCLUDED.region, lat=EXCLUDED.lat, lng=EXCLUDED.lng,
  contact=EXCLUDED.contact, url=EXCLUDED.url, updated_at=now()
RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q,
		p.Source, p.ExternalID, p.Title, p.Description, p.Category,
		orDefault(p.Status, "active"), p.Region, p.Lat, p.Lng, p.Contact, p.URL,
	).Scan(&id)
	return id, err
}

// UpsertResource inserts or updates a resource keyed by (source, external_id).
func (r *Repository) UpsertResource(ctx context.Context, res resource.Resource) (string, error) {
	const q = `
INSERT INTO resources (source, external_id, type, name, quantity, unit, status, region, lat, lng, contact)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (source, external_id) DO UPDATE SET
  type=EXCLUDED.type, name=EXCLUDED.name, quantity=EXCLUDED.quantity, unit=EXCLUDED.unit,
  status=EXCLUDED.status, region=EXCLUDED.region, lat=EXCLUDED.lat, lng=EXCLUDED.lng,
  contact=EXCLUDED.contact, updated_at=now()
RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q,
		res.Source, res.ExternalID, res.Type, res.Name, res.Quantity, res.Unit,
		orDefault(res.Status, "available"), res.Region, res.Lat, res.Lng, res.Contact,
	).Scan(&id)
	return id, err
}

// UpsertMissing inserts or updates a missing-person report keyed by (source, external_id).
func (r *Repository) UpsertMissing(ctx context.Context, m missing.Person) (string, error) {
	const q = `
INSERT INTO missing_persons (source, external_id, full_name, age, description, last_seen_region, last_seen_at, status, contact, photo_url)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
ON CONFLICT (source, external_id) DO UPDATE SET
  full_name=EXCLUDED.full_name, age=EXCLUDED.age, description=EXCLUDED.description,
  last_seen_region=EXCLUDED.last_seen_region, last_seen_at=EXCLUDED.last_seen_at,
  status=EXCLUDED.status, contact=EXCLUDED.contact, photo_url=EXCLUDED.photo_url, updated_at=now()
RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q,
		m.Source, m.ExternalID, m.FullName, m.Age, m.Description, m.LastSeenRegion,
		m.LastSeenAt, orDefault(m.Status, "missing"), m.Contact, m.PhotoURL,
	).Scan(&id)
	return id, err
}

// UpsertVolunteer inserts or updates a volunteer keyed by (source, external_id).
func (r *Repository) UpsertVolunteer(ctx context.Context, v volunteer.Volunteer) (string, error) {
	const q = `
INSERT INTO volunteers (source, external_id, full_name, skills, availability, region, contact, status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (source, external_id) DO UPDATE SET
  full_name=EXCLUDED.full_name, skills=EXCLUDED.skills, availability=EXCLUDED.availability,
  region=EXCLUDED.region, contact=EXCLUDED.contact, status=EXCLUDED.status, updated_at=now()
RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q,
		v.Source, v.ExternalID, v.FullName, v.Skills, v.Availability, v.Region,
		v.Contact, orDefault(v.Status, "available"),
	).Scan(&id)
	return id, err
}

// SyncSince returns every entity change with updated_at strictly after `since`,
// ordered oldest-first, capped at `limit`.
func (r *Repository) SyncSince(ctx context.Context, since time.Time, limit int) ([]syncdom.Change, error) {
	const q = `
SELECT entity, id::text, updated_at, data FROM (
  SELECT 'project'   AS entity, id, updated_at, to_jsonb(p) AS data FROM aid_projects p
  UNION ALL
  SELECT 'resource'  AS entity, id, updated_at, to_jsonb(r) AS data FROM resources r
  UNION ALL
  SELECT 'missing'   AS entity, id, updated_at, to_jsonb(m) AS data FROM missing_persons m
  UNION ALL
  SELECT 'volunteer' AS entity, id, updated_at, to_jsonb(v) AS data FROM volunteers v
) changes
WHERE updated_at > $1
ORDER BY updated_at ASC
LIMIT $2`
	rows, err := r.pool.Query(ctx, q, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	changes := make([]syncdom.Change, 0, limit)
	for rows.Next() {
		var c syncdom.Change
		if err := rows.Scan(&c.Entity, &c.ID, &c.UpdatedAt, &c.Data); err != nil {
			return nil, err
		}
		changes = append(changes, c)
	}
	return changes, rows.Err()
}

// InsertWebhookLog stores a raw inbound webhook payload for async processing.
func (r *Repository) InsertWebhookLog(ctx context.Context, source string, payload []byte) (string, error) {
	const q = `INSERT INTO webhooks_log (source, payload) VALUES ($1, $2) RETURNING id`
	var id string
	err := r.pool.QueryRow(ctx, q, source, payload).Scan(&id)
	return id, err
}

// CreateMagicToken issues a single-use login token valid for ttl.
func (r *Repository) CreateMagicToken(ctx context.Context, email string, ttl time.Duration) (string, time.Time, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token := hex.EncodeToString(buf)
	expiresAt := time.Now().Add(ttl)

	const q = `INSERT INTO magic_tokens (token, email, expires_at) VALUES ($1, $2, $3)`
	if _, err := r.pool.Exec(ctx, q, token, email, expiresAt); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// ConsumeMagicToken atomically validates and burns a magic token, returning the
// bound email. Returns ErrInvalidToken if it is unknown, already used, or expired.
func (r *Repository) ConsumeMagicToken(ctx context.Context, token string) (string, error) {
	const q = `
UPDATE magic_tokens SET used = true
WHERE token = $1 AND used = false AND expires_at > now()
RETURNING email`
	var email string
	err := r.pool.QueryRow(ctx, q, token).Scan(&email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrInvalidToken
	}
	return email, err
}

// Ping verifies database connectivity.
func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}
