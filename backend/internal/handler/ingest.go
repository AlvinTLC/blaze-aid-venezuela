package handler

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
)

// IngestResult is the common response for every ingest operation.
type IngestResult struct {
	ID     string `json:"id" doc:"Canonical hub id for the upserted entity"`
	Status string `json:"status" doc:"Always 'upserted' on success" example:"upserted"`
}

type ingestOutput struct {
	Body IngestResult
}

// ----- project -----

type ProjectInput struct {
	Body struct {
		Source      string   `json:"source" doc:"Origin feed/system" example:"twitter"`
		ExternalID  string   `json:"external_id" doc:"Stable id within the source"`
		Title       string   `json:"title" minLength:"1"`
		Description string   `json:"description,omitempty"`
		Category    string   `json:"category,omitempty"`
		Status      string   `json:"status,omitempty"`
		Region      string   `json:"region,omitempty"`
		Lat         *float64 `json:"lat,omitempty"`
		Lng         *float64 `json:"lng,omitempty"`
		Contact     string   `json:"contact,omitempty"`
		URL         string   `json:"url,omitempty"`
	}
}

func (h *Handler) IngestProject(ctx context.Context, in *ProjectInput) (*ingestOutput, error) {
	id, err := h.repo.UpsertProject(ctx, aidproject.AidProject{
		Source:      in.Body.Source,
		ExternalID:  in.Body.ExternalID,
		Title:       in.Body.Title,
		Description: in.Body.Description,
		Category:    in.Body.Category,
		Status:      in.Body.Status,
		Region:      in.Body.Region,
		Lat:         in.Body.Lat,
		Lng:         in.Body.Lng,
		Contact:     in.Body.Contact,
		URL:         in.Body.URL,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to ingest project", err)
	}
	return &ingestOutput{Body: IngestResult{ID: id, Status: "upserted"}}, nil
}

// ----- resource -----

type ResourceInput struct {
	Body struct {
		Source     string   `json:"source"`
		ExternalID string   `json:"external_id"`
		Type       string   `json:"type,omitempty"`
		Name       string   `json:"name" minLength:"1"`
		Quantity   float64  `json:"quantity,omitempty"`
		Unit       string   `json:"unit,omitempty"`
		Status     string   `json:"status,omitempty"`
		Region     string   `json:"region,omitempty"`
		Lat        *float64 `json:"lat,omitempty"`
		Lng        *float64 `json:"lng,omitempty"`
		Contact    string   `json:"contact,omitempty"`
	}
}

func (h *Handler) IngestResource(ctx context.Context, in *ResourceInput) (*ingestOutput, error) {
	id, err := h.repo.UpsertResource(ctx, resource.Resource{
		Source:     in.Body.Source,
		ExternalID: in.Body.ExternalID,
		Type:       in.Body.Type,
		Name:       in.Body.Name,
		Quantity:   in.Body.Quantity,
		Unit:       in.Body.Unit,
		Status:     in.Body.Status,
		Region:     in.Body.Region,
		Lat:        in.Body.Lat,
		Lng:        in.Body.Lng,
		Contact:    in.Body.Contact,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to ingest resource", err)
	}
	return &ingestOutput{Body: IngestResult{ID: id, Status: "upserted"}}, nil
}

// ----- missing -----

type MissingInput struct {
	Body struct {
		Source         string     `json:"source"`
		ExternalID     string     `json:"external_id"`
		FullName       string     `json:"full_name" minLength:"1"`
		Age            *int       `json:"age,omitempty"`
		Description    string     `json:"description,omitempty"`
		LastSeenRegion string     `json:"last_seen_region,omitempty"`
		Lat            *float64   `json:"lat,omitempty"`
		Lng            *float64   `json:"lng,omitempty"`
		LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
		Status         string     `json:"status,omitempty"`
		Contact        string     `json:"contact,omitempty"`
		PhotoURL       string     `json:"photo_url,omitempty"`
	}
}

func (h *Handler) IngestMissing(ctx context.Context, in *MissingInput) (*ingestOutput, error) {
	id, err := h.repo.UpsertMissing(ctx, missing.Person{
		Source:         in.Body.Source,
		ExternalID:     in.Body.ExternalID,
		FullName:       in.Body.FullName,
		Age:            in.Body.Age,
		Description:    in.Body.Description,
		LastSeenRegion: in.Body.LastSeenRegion,
		Lat:            in.Body.Lat,
		Lng:            in.Body.Lng,
		LastSeenAt:     in.Body.LastSeenAt,
		Status:         in.Body.Status,
		Contact:        in.Body.Contact,
		PhotoURL:       in.Body.PhotoURL,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to ingest missing person", err)
	}
	return &ingestOutput{Body: IngestResult{ID: id, Status: "upserted"}}, nil
}

// ----- volunteer -----

type VolunteerInput struct {
	Body struct {
		Source       string   `json:"source"`
		ExternalID   string   `json:"external_id"`
		FullName     string   `json:"full_name" minLength:"1"`
		Skills       []string `json:"skills,omitempty"`
		Availability string   `json:"availability,omitempty"`
		Region       string   `json:"region,omitempty"`
		Contact      string   `json:"contact,omitempty"`
		Status       string   `json:"status,omitempty"`
	}
}

func (h *Handler) IngestVolunteer(ctx context.Context, in *VolunteerInput) (*ingestOutput, error) {
	skills := in.Body.Skills
	if skills == nil {
		skills = []string{}
	}
	id, err := h.repo.UpsertVolunteer(ctx, volunteer.Volunteer{
		Source:       in.Body.Source,
		ExternalID:   in.Body.ExternalID,
		FullName:     in.Body.FullName,
		Skills:       skills,
		Availability: in.Body.Availability,
		Region:       in.Body.Region,
		Contact:      in.Body.Contact,
		Status:       in.Body.Status,
	})
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to ingest volunteer", err)
	}
	return &ingestOutput{Body: IngestResult{ID: id, Status: "upserted"}}, nil
}
