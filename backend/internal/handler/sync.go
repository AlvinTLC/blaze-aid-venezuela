package handler

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"

	syncdom "github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/domain/sync"
)

// SyncInput is the cursor-based query for GET /api/v1/sync.
type SyncInput struct {
	Since string `query:"since" doc:"RFC3339 timestamp; returns changes strictly after it. Empty = from epoch."`
	Limit int    `query:"limit" doc:"Max changes to return (1-1000)." default:"500" minimum:"1" maximum:"1000"`
}

// SyncOutput carries a page of changes plus the cursor to resume from.
type SyncOutput struct {
	Body struct {
		Changes []syncdom.Change `json:"changes"`
		Count   int              `json:"count"`
		Cursor  string           `json:"cursor" doc:"Pass as since on the next call (latest updated_at in this page)."`
	}
}

// Sync returns every entity change after the supplied cursor.
func (h *Handler) Sync(ctx context.Context, in *SyncInput) (*SyncOutput, error) {
	since := time.Time{}
	if in.Since != "" {
		parsed, err := time.Parse(time.RFC3339Nano, in.Since)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("invalid `since`; expected RFC3339", err)
		}
		since = parsed
	}

	limit := in.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}

	changes, err := h.repo.SyncSince(ctx, since, limit)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to read changes", err)
	}

	out := &SyncOutput{}
	out.Body.Changes = changes
	out.Body.Count = len(changes)
	out.Body.Cursor = in.Since
	if n := len(changes); n > 0 {
		out.Body.Cursor = changes[n-1].UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return out, nil
}
