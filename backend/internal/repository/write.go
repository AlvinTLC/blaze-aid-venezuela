package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
)

// Update structs carry pointer fields: nil = leave unchanged (PATCH), set = write
// (PUT passes all fields). SQL applies COALESCE($n, col) so nil keeps the column.

type ProjectUpdate struct {
	Title, Description, Category, Status, Region, Contact, URL *string
	Lat, Lng                                                  *float64
}

type ResourceUpdate struct {
	Type, Name, Unit, Status, Region, Contact *string
	Quantity                                  *float64
	Lat, Lng                                  *float64
}

type MissingUpdate struct {
	FullName, Description, LastSeenRegion, Status, Contact, PhotoURL *string
	Age                                                             *int
}

type VolunteerUpdate struct {
	FullName, Availability, Region, Contact, Status *string
	Skills                                          *[]string
}

var softDeletableTables = map[string]bool{
	"aid_projects": true, "resources": true, "missing_persons": true, "volunteers": true,
}

// softDelete marks a row deleted; ErrNotFound if it doesn't exist or is already gone.
func (r *Repository) softDelete(ctx context.Context, table, id string) error {
	if !softDeletableTables[table] {
		return errors.New("table not soft-deletable: " + table)
	}
	// table is whitelisted above, so it is safe to interpolate.
	q := "UPDATE " + table + " SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL RETURNING id"
	var got string
	err := r.pool.QueryRow(ctx, q, id).Scan(&got)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *Repository) DeleteProject(ctx context.Context, id string) error {
	return r.softDelete(ctx, "aid_projects", id)
}
func (r *Repository) DeleteResource(ctx context.Context, id string) error {
	return r.softDelete(ctx, "resources", id)
}
func (r *Repository) DeleteMissing(ctx context.Context, id string) error {
	return r.softDelete(ctx, "missing_persons", id)
}
func (r *Repository) DeleteVolunteer(ctx context.Context, id string) error {
	return r.softDelete(ctx, "volunteers", id)
}

// touched returns ErrNotFound when an update matched no active row.
func touched(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (r *Repository) UpdateProject(ctx context.Context, id string, u ProjectUpdate) (aidproject.AidProject, error) {
	const q = `
UPDATE aid_projects SET
  title=COALESCE($2,title), description=COALESCE($3,description), category=COALESCE($4,category),
  status=COALESCE($5,status), region=COALESCE($6,region), contact=COALESCE($7,contact),
  url=COALESCE($8,url), lat=COALESCE($9,lat), lng=COALESCE($10,lng), updated_at=now()
WHERE id=$1 AND deleted_at IS NULL RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id, u.Title, u.Description, u.Category, u.Status, u.Region,
		u.Contact, u.URL, u.Lat, u.Lng).Scan(&got)
	if err != nil {
		return aidproject.AidProject{}, touched(err)
	}
	return r.GetProject(ctx, id)
}

func (r *Repository) UpdateResource(ctx context.Context, id string, u ResourceUpdate) (resource.Resource, error) {
	const q = `
UPDATE resources SET
  type=COALESCE($2,type), name=COALESCE($3,name), unit=COALESCE($4,unit), status=COALESCE($5,status),
  region=COALESCE($6,region), contact=COALESCE($7,contact), quantity=COALESCE($8,quantity),
  lat=COALESCE($9,lat), lng=COALESCE($10,lng), updated_at=now()
WHERE id=$1 AND deleted_at IS NULL RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id, u.Type, u.Name, u.Unit, u.Status, u.Region, u.Contact,
		u.Quantity, u.Lat, u.Lng).Scan(&got)
	if err != nil {
		return resource.Resource{}, touched(err)
	}
	return r.GetResource(ctx, id)
}

func (r *Repository) UpdateMissing(ctx context.Context, id string, u MissingUpdate) (missing.Person, error) {
	const q = `
UPDATE missing_persons SET
  full_name=COALESCE($2,full_name), description=COALESCE($3,description),
  last_seen_region=COALESCE($4,last_seen_region), status=COALESCE($5,status),
  contact=COALESCE($6,contact), photo_url=COALESCE($7,photo_url), age=COALESCE($8,age), updated_at=now()
WHERE id=$1 AND deleted_at IS NULL RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id, u.FullName, u.Description, u.LastSeenRegion, u.Status,
		u.Contact, u.PhotoURL, u.Age).Scan(&got)
	if err != nil {
		return missing.Person{}, touched(err)
	}
	return r.GetMissing(ctx, id)
}

func (r *Repository) UpdateVolunteer(ctx context.Context, id string, u VolunteerUpdate) (volunteer.Volunteer, error) {
	const q = `
UPDATE volunteers SET
  full_name=COALESCE($2,full_name), availability=COALESCE($3,availability), region=COALESCE($4,region),
  contact=COALESCE($5,contact), status=COALESCE($6,status), skills=COALESCE($7,skills), updated_at=now()
WHERE id=$1 AND deleted_at IS NULL RETURNING id`
	var got string
	err := r.pool.QueryRow(ctx, q, id, u.FullName, u.Availability, u.Region, u.Contact, u.Status,
		u.Skills).Scan(&got)
	if err != nil {
		return volunteer.Volunteer{}, touched(err)
	}
	return r.GetVolunteer(ctx, id)
}
