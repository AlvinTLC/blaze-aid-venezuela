package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/auth"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// ListQuery holds the common pagination + filter query params for list endpoints.
// Authorization is optional: a valid bearer token unlocks PII fields (contact).
type ListQuery struct {
	Authorization string `header:"Authorization" required:"false"`
	Region        string `query:"region" doc:"Filter by region"`
	Status        string `query:"status" doc:"Filter by status"`
	Q             string `query:"q" doc:"Free-text search"`
	Limit         int    `query:"limit" default:"20" minimum:"1" maximum:"100"`
	Offset        int    `query:"offset" default:"0" minimum:"0"`
}

func (q ListQuery) params(extra string) repository.ListParams {
	return repository.ListParams{
		Region: q.Region, Status: q.Status, Extra: extra, Q: q.Q, Limit: q.Limit, Offset: q.Offset,
	}
}

// authed reports whether the Authorization header carries a valid session JWT.
func (h *Handler) authed(authz string) bool {
	const p = "Bearer "
	if !strings.HasPrefix(authz, p) {
		return false
	}
	_, err := auth.ParseJWT(h.jwtSecret, strings.TrimPrefix(authz, p))
	return err == nil
}

func page(limit, offset int) int {
	limit = clampLimit(limit)
	if limit == 0 {
		return 1
	}
	return offset/limit + 1
}

// ----- projects -----

type ListProjectsInput struct {
	ListQuery
	Category string `query:"category" doc:"Filter by category"`
}

type ListProjectsOutput struct {
	Body struct {
		Items  []aidproject.AidProject `json:"items"`
		Total  int                     `json:"total"`
		Limit  int                     `json:"limit"`
		Offset int                     `json:"offset"`
		Page   int                     `json:"page"`
	}
}

func (h *Handler) ListProjects(ctx context.Context, in *ListProjectsInput) (*ListProjectsOutput, error) {
	items, total, err := h.repo.ListProjects(ctx, in.params(in.Category))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list projects", err)
	}
	if !h.authed(in.Authorization) {
		for i := range items {
			items[i].Contact = ""
		}
	}
	out := &ListProjectsOutput{}
	out.Body.Items, out.Body.Total = items, total
	out.Body.Limit, out.Body.Offset, out.Body.Page = clampLimit(in.Limit), in.Offset, page(in.Limit, in.Offset)
	return out, nil
}

type ProjectByIDInput struct {
	ID            string `path:"id" format:"uuid"`
	Authorization string `header:"Authorization" required:"false"`
}
type ProjectOutput struct{ Body aidproject.AidProject }

func (h *Handler) GetProject(ctx context.Context, in *ProjectByIDInput) (*ProjectOutput, error) {
	item, err := h.repo.GetProject(ctx, in.ID)
	if err != nil {
		return nil, notFoundOr(err, "project")
	}
	if !h.authed(in.Authorization) {
		item.Contact = ""
	}
	return &ProjectOutput{Body: item}, nil
}

// ----- resources -----

type ListResourcesInput struct {
	ListQuery
	Type string `query:"type" doc:"Filter by resource type"`
}

type ListResourcesOutput struct {
	Body struct {
		Items  []resource.Resource `json:"items"`
		Total  int                 `json:"total"`
		Limit  int                 `json:"limit"`
		Offset int                 `json:"offset"`
		Page   int                 `json:"page"`
	}
}

func (h *Handler) ListResources(ctx context.Context, in *ListResourcesInput) (*ListResourcesOutput, error) {
	items, total, err := h.repo.ListResources(ctx, in.params(in.Type))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list resources", err)
	}
	if !h.authed(in.Authorization) {
		for i := range items {
			items[i].Contact = ""
		}
	}
	out := &ListResourcesOutput{}
	out.Body.Items, out.Body.Total = items, total
	out.Body.Limit, out.Body.Offset, out.Body.Page = clampLimit(in.Limit), in.Offset, page(in.Limit, in.Offset)
	return out, nil
}

type ResourceByIDInput struct {
	ID            string `path:"id" format:"uuid"`
	Authorization string `header:"Authorization" required:"false"`
}
type ResourceOutput struct{ Body resource.Resource }

func (h *Handler) GetResource(ctx context.Context, in *ResourceByIDInput) (*ResourceOutput, error) {
	item, err := h.repo.GetResource(ctx, in.ID)
	if err != nil {
		return nil, notFoundOr(err, "resource")
	}
	if !h.authed(in.Authorization) {
		item.Contact = ""
	}
	return &ResourceOutput{Body: item}, nil
}

// ----- missing -----

type ListMissingInput struct {
	ListQuery
}

type ListMissingOutput struct {
	Body struct {
		Items  []missing.Person `json:"items"`
		Total  int              `json:"total"`
		Limit  int              `json:"limit"`
		Offset int              `json:"offset"`
		Page   int              `json:"page"`
	}
}

func (h *Handler) ListMissing(ctx context.Context, in *ListMissingInput) (*ListMissingOutput, error) {
	items, total, err := h.repo.ListMissing(ctx, in.params(""))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list missing persons", err)
	}
	if !h.authed(in.Authorization) {
		for i := range items {
			items[i].Contact = ""
		}
	}
	out := &ListMissingOutput{}
	out.Body.Items, out.Body.Total = items, total
	out.Body.Limit, out.Body.Offset, out.Body.Page = clampLimit(in.Limit), in.Offset, page(in.Limit, in.Offset)
	return out, nil
}

type MissingByIDInput struct {
	ID            string `path:"id" format:"uuid"`
	Authorization string `header:"Authorization" required:"false"`
}
type MissingOutput struct{ Body missing.Person }

func (h *Handler) GetMissing(ctx context.Context, in *MissingByIDInput) (*MissingOutput, error) {
	item, err := h.repo.GetMissing(ctx, in.ID)
	if err != nil {
		return nil, notFoundOr(err, "missing person")
	}
	if !h.authed(in.Authorization) {
		item.Contact = ""
	}
	return &MissingOutput{Body: item}, nil
}

// ----- volunteers -----

type ListVolunteersInput struct {
	ListQuery
	Skill string `query:"skill" doc:"Filter by a skill the volunteer has"`
}

type ListVolunteersOutput struct {
	Body struct {
		Items  []volunteer.Volunteer `json:"items"`
		Total  int                   `json:"total"`
		Limit  int                   `json:"limit"`
		Offset int                   `json:"offset"`
		Page   int                   `json:"page"`
	}
}

func (h *Handler) ListVolunteers(ctx context.Context, in *ListVolunteersInput) (*ListVolunteersOutput, error) {
	items, total, err := h.repo.ListVolunteers(ctx, in.params(in.Skill))
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to list volunteers", err)
	}
	if !h.authed(in.Authorization) {
		for i := range items {
			items[i].Contact = ""
		}
	}
	out := &ListVolunteersOutput{}
	out.Body.Items, out.Body.Total = items, total
	out.Body.Limit, out.Body.Offset, out.Body.Page = clampLimit(in.Limit), in.Offset, page(in.Limit, in.Offset)
	return out, nil
}

type VolunteerByIDInput struct {
	ID            string `path:"id" format:"uuid"`
	Authorization string `header:"Authorization" required:"false"`
}
type VolunteerOutput struct{ Body volunteer.Volunteer }

func (h *Handler) GetVolunteer(ctx context.Context, in *VolunteerByIDInput) (*VolunteerOutput, error) {
	item, err := h.repo.GetVolunteer(ctx, in.ID)
	if err != nil {
		return nil, notFoundOr(err, "volunteer")
	}
	if !h.authed(in.Authorization) {
		item.Contact = ""
	}
	return &VolunteerOutput{Body: item}, nil
}

// helpers

func clampLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func notFoundOr(err error, what string) error {
	if errors.Is(err, repository.ErrNotFound) {
		return huma.Error404NotFound(what + " not found")
	}
	return huma.Error500InternalServerError("failed to load "+what, err)
}
