package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
)

// whereBuilder accumulates parameterized WHERE conditions.
type whereBuilder struct {
	conds []string
	args  []any
}

// newActiveWhere starts a filter that excludes soft-deleted rows.
func newActiveWhere() *whereBuilder {
	return &whereBuilder{conds: []string{"deleted_at IS NULL"}}
}

func (w *whereBuilder) eq(col, val string) {
	if val == "" {
		return
	}
	w.args = append(w.args, val)
	w.conds = append(w.conds, fmt.Sprintf("%s = $%d", col, len(w.args)))
}

// ilikeAny matches val (case-insensitive substring) against any of the columns.
func (w *whereBuilder) ilikeAny(val string, cols ...string) {
	if val == "" {
		return
	}
	w.args = append(w.args, "%"+val+"%")
	n := len(w.args)
	parts := make([]string, len(cols))
	for i, c := range cols {
		parts[i] = fmt.Sprintf("%s ILIKE $%d", c, n)
	}
	w.conds = append(w.conds, "("+strings.Join(parts, " OR ")+")")
}

// arrayContains matches when the text[] column contains val.
func (w *whereBuilder) arrayContains(col, val string) {
	if val == "" {
		return
	}
	w.args = append(w.args, val)
	w.conds = append(w.conds, fmt.Sprintf("$%d = ANY(%s)", len(w.args), col))
}

// tsRange adds created_at >= from and/or <= to.
func (w *whereBuilder) tsRange(col string, from, to *time.Time) {
	if from != nil {
		w.args = append(w.args, *from)
		w.conds = append(w.conds, fmt.Sprintf("%s >= $%d", col, len(w.args)))
	}
	if to != nil {
		w.args = append(w.args, *to)
		w.conds = append(w.conds, fmt.Sprintf("%s <= $%d", col, len(w.args)))
	}
}

// nearMe filters rows within radiusKm of (lat,lng) using the haversine formula
// (no PostGIS needed). Rows without coordinates are excluded.
func (w *whereBuilder) nearMe(latCol, lngCol string, lat, lng *float64, radiusKm float64) {
	if lat == nil || lng == nil || radiusKm <= 0 {
		return
	}
	w.args = append(w.args, *lat)
	a := len(w.args)
	w.args = append(w.args, *lng)
	b := len(w.args)
	w.args = append(w.args, radiusKm)
	r := len(w.args)
	w.conds = append(w.conds, fmt.Sprintf(
		"(%s IS NOT NULL AND %s IS NOT NULL AND 6371 * acos(LEAST(1, "+
			"cos(radians($%d))*cos(radians(%s))*cos(radians(%s)-radians($%d))"+
			"+sin(radians($%d))*sin(radians(%s)))) <= $%d)",
		latCol, lngCol, a, latCol, lngCol, b, a, latCol, r))
}

func (w *whereBuilder) clause() string {
	if len(w.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(w.conds, " AND ")
}

// ----- projects -----

func (r *Repository) ListProjects(ctx context.Context, p ListParams) ([]aidproject.AidProject, int, error) {
	w := newActiveWhere()
	w.eq("region", p.Region)
	w.eq("status", p.Status)
	w.eq("category", p.Extra)
	w.ilikeAny(p.Q, "title", "description")
	w.tsRange("created_at", p.From, p.To)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM aid_projects`+w.clause(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := p.clamp()
	q := `SELECT id, source, external_id, title, description, category, status, region,
	             lat, lng, contact, url, created_at, updated_at
	      FROM aid_projects` + w.clause() +
		fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2)

	rows, err := r.pool.Query(ctx, q, append(w.args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]aidproject.AidProject, 0, limit)
	for rows.Next() {
		var p aidproject.AidProject
		if err := rows.Scan(&p.ID, &p.Source, &p.ExternalID, &p.Title, &p.Description, &p.Category,
			&p.Status, &p.Region, &p.Lat, &p.Lng, &p.Contact, &p.URL, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, p)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetProject(ctx context.Context, id string) (aidproject.AidProject, error) {
	const q = `SELECT id, source, external_id, title, description, category, status, region,
	                  lat, lng, contact, url, created_at, updated_at
	           FROM aid_projects WHERE id = $1 AND deleted_at IS NULL`
	var p aidproject.AidProject
	err := r.pool.QueryRow(ctx, q, id).Scan(&p.ID, &p.Source, &p.ExternalID, &p.Title, &p.Description,
		&p.Category, &p.Status, &p.Region, &p.Lat, &p.Lng, &p.Contact, &p.URL, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return aidproject.AidProject{}, ErrNotFound
	}
	return p, err
}

// ----- resources -----

func (r *Repository) ListResources(ctx context.Context, p ListParams) ([]resource.Resource, int, error) {
	w := newActiveWhere()
	w.eq("region", p.Region)
	w.eq("status", p.Status)
	w.eq("type", p.Extra)
	w.ilikeAny(p.Q, "name", "type")
	w.tsRange("created_at", p.From, p.To)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM resources`+w.clause(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := p.clamp()
	q := `SELECT id, source, external_id, type, name, quantity, unit, status, region,
	             lat, lng, contact, created_at, updated_at
	      FROM resources` + w.clause() +
		fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2)

	rows, err := r.pool.Query(ctx, q, append(w.args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]resource.Resource, 0, limit)
	for rows.Next() {
		var res resource.Resource
		if err := rows.Scan(&res.ID, &res.Source, &res.ExternalID, &res.Type, &res.Name, &res.Quantity,
			&res.Unit, &res.Status, &res.Region, &res.Lat, &res.Lng, &res.Contact, &res.CreatedAt, &res.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, res)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetResource(ctx context.Context, id string) (resource.Resource, error) {
	const q = `SELECT id, source, external_id, type, name, quantity, unit, status, region,
	                  lat, lng, contact, created_at, updated_at
	           FROM resources WHERE id = $1 AND deleted_at IS NULL`
	var res resource.Resource
	err := r.pool.QueryRow(ctx, q, id).Scan(&res.ID, &res.Source, &res.ExternalID, &res.Type, &res.Name,
		&res.Quantity, &res.Unit, &res.Status, &res.Region, &res.Lat, &res.Lng, &res.Contact, &res.CreatedAt, &res.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return resource.Resource{}, ErrNotFound
	}
	return res, err
}

// ----- missing -----

func (r *Repository) ListMissing(ctx context.Context, p ListParams) ([]missing.Person, int, error) {
	w := newActiveWhere()
	w.eq("last_seen_region", p.Region)
	w.eq("status", p.Status)
	w.ilikeAny(p.Q, "full_name", "description")
	w.tsRange("created_at", p.From, p.To)
	w.nearMe("lat", "lng", p.Lat, p.Lng, p.RadiusKm)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM missing_persons`+w.clause(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := p.clamp()
	q := `SELECT id, source, external_id, full_name, age, description, last_seen_region, lat, lng, last_seen_at,
	             status, contact, photo_url, created_at, updated_at
	      FROM missing_persons` + w.clause() +
		fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2)

	rows, err := r.pool.Query(ctx, q, append(w.args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]missing.Person, 0, limit)
	for rows.Next() {
		var m missing.Person
		if err := rows.Scan(&m.ID, &m.Source, &m.ExternalID, &m.FullName, &m.Age, &m.Description,
			&m.LastSeenRegion, &m.Lat, &m.Lng, &m.LastSeenAt, &m.Status, &m.Contact, &m.PhotoURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, m)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetMissing(ctx context.Context, id string) (missing.Person, error) {
	const q = `SELECT id, source, external_id, full_name, age, description, last_seen_region, lat, lng, last_seen_at,
	                  status, contact, photo_url, created_at, updated_at
	           FROM missing_persons WHERE id = $1 AND deleted_at IS NULL`
	var m missing.Person
	err := r.pool.QueryRow(ctx, q, id).Scan(&m.ID, &m.Source, &m.ExternalID, &m.FullName, &m.Age, &m.Description,
		&m.LastSeenRegion, &m.Lat, &m.Lng, &m.LastSeenAt, &m.Status, &m.Contact, &m.PhotoURL, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return missing.Person{}, ErrNotFound
	}
	return m, err
}

// ----- volunteers -----

func (r *Repository) ListVolunteers(ctx context.Context, p ListParams) ([]volunteer.Volunteer, int, error) {
	w := newActiveWhere()
	w.eq("region", p.Region)
	w.eq("status", p.Status)
	w.arrayContains("skills", p.Extra)
	w.ilikeAny(p.Q, "full_name")
	w.tsRange("created_at", p.From, p.To)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM volunteers`+w.clause(), w.args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit, offset := p.clamp()
	q := `SELECT id, source, external_id, full_name, skills, availability, region, contact, status,
	             created_at, updated_at
	      FROM volunteers` + w.clause() +
		fmt.Sprintf(" ORDER BY updated_at DESC LIMIT $%d OFFSET $%d", len(w.args)+1, len(w.args)+2)

	rows, err := r.pool.Query(ctx, q, append(w.args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]volunteer.Volunteer, 0, limit)
	for rows.Next() {
		var v volunteer.Volunteer
		if err := rows.Scan(&v.ID, &v.Source, &v.ExternalID, &v.FullName, &v.Skills, &v.Availability,
			&v.Region, &v.Contact, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	return items, total, rows.Err()
}

func (r *Repository) GetVolunteer(ctx context.Context, id string) (volunteer.Volunteer, error) {
	const q = `SELECT id, source, external_id, full_name, skills, availability, region, contact, status,
	                  created_at, updated_at
	           FROM volunteers WHERE id = $1 AND deleted_at IS NULL`
	var v volunteer.Volunteer
	err := r.pool.QueryRow(ctx, q, id).Scan(&v.ID, &v.Source, &v.ExternalID, &v.FullName, &v.Skills,
		&v.Availability, &v.Region, &v.Contact, &v.Status, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return volunteer.Volunteer{}, ErrNotFound
	}
	return v, err
}
