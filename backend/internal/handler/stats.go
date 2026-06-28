package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/aidproject"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/missing"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/resource"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/volunteer"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// StatsData is the dashboard aggregate. NOTE: /stats is the only endpoint that
// wraps its payload in `data`; the catalog endpoints return {items,total,...}.
type StatsData struct {
	Counts   map[string]int            `json:"counts"`
	ByStatus map[string]map[string]int `json:"by_status"`
	ByRegion map[string]map[string]int `json:"by_region"`
	Recent   RecentEntities            `json:"recent"`
	Timeline Timeline                  `json:"timeline"`
}

type RecentEntities struct {
	Projects   []aidproject.AidProject `json:"projects"`
	Resources  []resource.Resource     `json:"resources"`
	Missing    []missing.Person        `json:"missing"`
	Volunteers []volunteer.Volunteer   `json:"volunteers"`
}

type Timeline struct {
	Labels   []string  `json:"labels"`
	Datasets []Dataset `json:"datasets"`
}

type Dataset struct {
	Label string `json:"label"`
	Data  []int  `json:"data"`
}

type StatsOutput struct {
	Body struct {
		Data StatsData `json:"data"`
	}
}

// Stats returns aggregate dashboard data (public). Recent rows have contact PII
// redacted since the endpoint is unauthenticated.
func (h *Handler) Stats(ctx context.Context, _ *struct{}) (*StatsOutput, error) {
	counts, err := h.repo.StatsCounts(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to compute counts", err)
	}

	byStatus := map[string]map[string]int{}
	byRegion := map[string]map[string]int{}
	statusTables := map[string]string{
		"projects": "aid_projects", "resources": "resources",
		"missing": "missing_persons", "volunteers": "volunteers",
	}
	regionCols := map[string]string{
		"projects": "region", "resources": "region",
		"missing": "last_seen_region", "volunteers": "region",
	}
	for key, table := range statusTables {
		if byStatus[key], err = h.repo.GroupCount(ctx, table, "status"); err != nil {
			return nil, huma.Error500InternalServerError("failed status breakdown", err)
		}
		if byRegion[key], err = h.repo.GroupCount(ctx, table, regionCols[key]); err != nil {
			return nil, huma.Error500InternalServerError("failed region breakdown", err)
		}
	}

	recent, err := h.recentEntities(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed recent", err)
	}

	labels, data, err := h.repo.EventsTimeline(ctx, 7)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed timeline", err)
	}

	out := &StatsOutput{}
	out.Body.Data = StatsData{
		Counts:   counts,
		ByStatus: byStatus,
		ByRegion: byRegion,
		Recent:   recent,
		Timeline: Timeline{Labels: labels, Datasets: []Dataset{{Label: "events", Data: data}}},
	}
	return out, nil
}

func (h *Handler) recentEntities(ctx context.Context) (RecentEntities, error) {
	const n = 5
	var r RecentEntities

	projects, _, err := h.repo.ListProjects(ctx, repository.ListParams{Limit: n})
	if err != nil {
		return r, err
	}
	resources, _, err := h.repo.ListResources(ctx, repository.ListParams{Limit: n})
	if err != nil {
		return r, err
	}
	miss, _, err := h.repo.ListMissing(ctx, repository.ListParams{Limit: n})
	if err != nil {
		return r, err
	}
	vols, _, err := h.repo.ListVolunteers(ctx, repository.ListParams{Limit: n})
	if err != nil {
		return r, err
	}

	// Public endpoint: redact contact PII from recent rows.
	for i := range projects {
		projects[i].Contact = ""
	}
	for i := range resources {
		resources[i].Contact = ""
	}
	for i := range miss {
		miss[i].Contact = ""
	}
	for i := range vols {
		vols[i].Contact = ""
	}

	r.Projects, r.Resources, r.Missing, r.Volunteers = projects, resources, miss, vols
	return r, nil
}
