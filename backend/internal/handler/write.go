package handler

import (
	"context"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// DeleteByIDInput is shared by every soft-delete endpoint.
type DeleteByIDInput struct {
	ID string `path:"id" format:"uuid"`
}

// NoContent is an empty 204 response body.
type NoContent struct{}

// ----- projects -----

type PatchProjectInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Title       *string  `json:"title,omitempty"`
		Description *string  `json:"description,omitempty"`
		Category    *string  `json:"category,omitempty"`
		Status      *string  `json:"status,omitempty"`
		Region      *string  `json:"region,omitempty"`
		Contact     *string  `json:"contact,omitempty"`
		URL         *string  `json:"url,omitempty"`
		Lat         *float64 `json:"lat,omitempty"`
		Lng         *float64 `json:"lng,omitempty"`
	}
}

func (h *Handler) PatchProject(ctx context.Context, in *PatchProjectInput) (*ProjectOutput, error) {
	item, err := h.repo.UpdateProject(ctx, in.ID, repository.ProjectUpdate{
		Title: in.Body.Title, Description: in.Body.Description, Category: in.Body.Category,
		Status: in.Body.Status, Region: in.Body.Region, Contact: in.Body.Contact, URL: in.Body.URL,
		Lat: in.Body.Lat, Lng: in.Body.Lng,
	})
	if err != nil {
		return nil, notFoundOr(err, "project")
	}
	return &ProjectOutput{Body: item}, nil
}

func (h *Handler) DeleteProject(ctx context.Context, in *DeleteByIDInput) (*NoContent, error) {
	if err := h.repo.DeleteProject(ctx, in.ID); err != nil {
		return nil, notFoundOr(err, "project")
	}
	return &NoContent{}, nil
}

type PutProjectInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Title       string   `json:"title" minLength:"1"`
		Description string   `json:"description,omitempty"`
		Category    string   `json:"category,omitempty"`
		Status      string   `json:"status,omitempty"`
		Region      string   `json:"region,omitempty"`
		Contact     string   `json:"contact,omitempty"`
		URL         string   `json:"url,omitempty"`
		Lat         *float64 `json:"lat,omitempty"`
		Lng         *float64 `json:"lng,omitempty"`
	}
}

func (h *Handler) PutProject(ctx context.Context, in *PutProjectInput) (*ProjectOutput, error) {
	b := &in.Body
	item, err := h.repo.UpdateProject(ctx, in.ID, repository.ProjectUpdate{
		Title: &b.Title, Description: &b.Description, Category: &b.Category, Status: &b.Status,
		Region: &b.Region, Contact: &b.Contact, URL: &b.URL, Lat: b.Lat, Lng: b.Lng,
	})
	if err != nil {
		return nil, notFoundOr(err, "project")
	}
	return &ProjectOutput{Body: item}, nil
}

// ----- resources -----

type PatchResourceInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Type     *string  `json:"type,omitempty"`
		Name     *string  `json:"name,omitempty"`
		Unit     *string  `json:"unit,omitempty"`
		Status   *string  `json:"status,omitempty"`
		Region   *string  `json:"region,omitempty"`
		Contact  *string  `json:"contact,omitempty"`
		Quantity *float64 `json:"quantity,omitempty"`
		Lat      *float64 `json:"lat,omitempty"`
		Lng      *float64 `json:"lng,omitempty"`
	}
}

func (h *Handler) PatchResource(ctx context.Context, in *PatchResourceInput) (*ResourceOutput, error) {
	item, err := h.repo.UpdateResource(ctx, in.ID, repository.ResourceUpdate{
		Type: in.Body.Type, Name: in.Body.Name, Unit: in.Body.Unit, Status: in.Body.Status,
		Region: in.Body.Region, Contact: in.Body.Contact, Quantity: in.Body.Quantity,
		Lat: in.Body.Lat, Lng: in.Body.Lng,
	})
	if err != nil {
		return nil, notFoundOr(err, "resource")
	}
	return &ResourceOutput{Body: item}, nil
}

func (h *Handler) DeleteResource(ctx context.Context, in *DeleteByIDInput) (*NoContent, error) {
	if err := h.repo.DeleteResource(ctx, in.ID); err != nil {
		return nil, notFoundOr(err, "resource")
	}
	return &NoContent{}, nil
}

type PutResourceInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		Type     string   `json:"type,omitempty"`
		Name     string   `json:"name" minLength:"1"`
		Unit     string   `json:"unit,omitempty"`
		Status   string   `json:"status,omitempty"`
		Region   string   `json:"region,omitempty"`
		Contact  string   `json:"contact,omitempty"`
		Quantity *float64 `json:"quantity,omitempty"`
		Lat      *float64 `json:"lat,omitempty"`
		Lng      *float64 `json:"lng,omitempty"`
	}
}

func (h *Handler) PutResource(ctx context.Context, in *PutResourceInput) (*ResourceOutput, error) {
	b := &in.Body
	item, err := h.repo.UpdateResource(ctx, in.ID, repository.ResourceUpdate{
		Type: &b.Type, Name: &b.Name, Unit: &b.Unit, Status: &b.Status,
		Region: &b.Region, Contact: &b.Contact, Quantity: b.Quantity, Lat: b.Lat, Lng: b.Lng,
	})
	if err != nil {
		return nil, notFoundOr(err, "resource")
	}
	return &ResourceOutput{Body: item}, nil
}

// ----- missing -----

type PatchMissingInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		FullName       *string `json:"full_name,omitempty"`
		Description    *string `json:"description,omitempty"`
		LastSeenRegion *string `json:"last_seen_region,omitempty"`
		Status         *string `json:"status,omitempty"`
		Contact        *string `json:"contact,omitempty"`
		PhotoURL       *string `json:"photo_url,omitempty"`
		Age            *int    `json:"age,omitempty"`
	}
}

func (h *Handler) PatchMissing(ctx context.Context, in *PatchMissingInput) (*MissingOutput, error) {
	item, err := h.repo.UpdateMissing(ctx, in.ID, repository.MissingUpdate{
		FullName: in.Body.FullName, Description: in.Body.Description, LastSeenRegion: in.Body.LastSeenRegion,
		Status: in.Body.Status, Contact: in.Body.Contact, PhotoURL: in.Body.PhotoURL, Age: in.Body.Age,
	})
	if err != nil {
		return nil, notFoundOr(err, "missing person")
	}
	return &MissingOutput{Body: item}, nil
}

func (h *Handler) DeleteMissing(ctx context.Context, in *DeleteByIDInput) (*NoContent, error) {
	if err := h.repo.DeleteMissing(ctx, in.ID); err != nil {
		return nil, notFoundOr(err, "missing person")
	}
	return &NoContent{}, nil
}

type PutMissingInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		FullName       string `json:"full_name" minLength:"1"`
		Description    string `json:"description,omitempty"`
		LastSeenRegion string `json:"last_seen_region,omitempty"`
		Status         string `json:"status,omitempty"`
		Contact        string `json:"contact,omitempty"`
		PhotoURL       string `json:"photo_url,omitempty"`
		Age            *int   `json:"age,omitempty"`
	}
}

func (h *Handler) PutMissing(ctx context.Context, in *PutMissingInput) (*MissingOutput, error) {
	b := &in.Body
	item, err := h.repo.UpdateMissing(ctx, in.ID, repository.MissingUpdate{
		FullName: &b.FullName, Description: &b.Description, LastSeenRegion: &b.LastSeenRegion,
		Status: &b.Status, Contact: &b.Contact, PhotoURL: &b.PhotoURL, Age: b.Age,
	})
	if err != nil {
		return nil, notFoundOr(err, "missing person")
	}
	return &MissingOutput{Body: item}, nil
}

// ----- volunteers -----

type PatchVolunteerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		FullName     *string   `json:"full_name,omitempty"`
		Availability *string   `json:"availability,omitempty"`
		Region       *string   `json:"region,omitempty"`
		Contact      *string   `json:"contact,omitempty"`
		Status       *string   `json:"status,omitempty"`
		Skills       *[]string `json:"skills,omitempty"`
	}
}

func (h *Handler) PatchVolunteer(ctx context.Context, in *PatchVolunteerInput) (*VolunteerOutput, error) {
	item, err := h.repo.UpdateVolunteer(ctx, in.ID, repository.VolunteerUpdate{
		FullName: in.Body.FullName, Availability: in.Body.Availability, Region: in.Body.Region,
		Contact: in.Body.Contact, Status: in.Body.Status, Skills: in.Body.Skills,
	})
	if err != nil {
		return nil, notFoundOr(err, "volunteer")
	}
	return &VolunteerOutput{Body: item}, nil
}

func (h *Handler) DeleteVolunteer(ctx context.Context, in *DeleteByIDInput) (*NoContent, error) {
	if err := h.repo.DeleteVolunteer(ctx, in.ID); err != nil {
		return nil, notFoundOr(err, "volunteer")
	}
	return &NoContent{}, nil
}

type PutVolunteerInput struct {
	ID   string `path:"id" format:"uuid"`
	Body struct {
		FullName     string   `json:"full_name" minLength:"1"`
		Availability string   `json:"availability,omitempty"`
		Region       string   `json:"region,omitempty"`
		Contact      string   `json:"contact,omitempty"`
		Status       string   `json:"status,omitempty"`
		Skills       []string `json:"skills,omitempty"`
	}
}

func (h *Handler) PutVolunteer(ctx context.Context, in *PutVolunteerInput) (*VolunteerOutput, error) {
	b := &in.Body
	skills := b.Skills
	if skills == nil {
		skills = []string{}
	}
	item, err := h.repo.UpdateVolunteer(ctx, in.ID, repository.VolunteerUpdate{
		FullName: &b.FullName, Availability: &b.Availability, Region: &b.Region,
		Contact: &b.Contact, Status: &b.Status, Skills: &skills,
	})
	if err != nil {
		return nil, notFoundOr(err, "volunteer")
	}
	return &VolunteerOutput{Body: item}, nil
}
