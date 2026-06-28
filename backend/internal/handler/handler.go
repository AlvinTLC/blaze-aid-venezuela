package handler

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// Handler holds dependencies shared by every operation.
type Handler struct {
	repo       *repository.Repository
	jwtSecret  string
	production bool
	logger     *slog.Logger
}

// New constructs a Handler. When production is true, sensitive stub behaviour
// (e.g. returning the magic-login token in the response) is disabled.
func New(repo *repository.Repository, jwtSecret string, production bool, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, jwtSecret: jwtSecret, production: production, logger: logger}
}

// Register wires every P0 operation onto the Huma API.
func (h *Handler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "ingest-project",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/project",
		Summary:     "Ingest an aid project",
		Tags:        []string{"ingest"},
	}, h.IngestProject)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-resource",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/resource",
		Summary:     "Ingest a resource",
		Tags:        []string{"ingest"},
	}, h.IngestResource)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-missing",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/missing",
		Summary:     "Ingest a missing-person report",
		Tags:        []string{"ingest"},
	}, h.IngestMissing)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-volunteer",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/volunteer",
		Summary:     "Ingest a volunteer",
		Tags:        []string{"ingest"},
	}, h.IngestVolunteer)

	huma.Register(api, huma.Operation{
		OperationID: "sync",
		Method:      http.MethodGet,
		Path:        "/api/v1/sync",
		Summary:     "Pull entity changes since a cursor",
		Tags:        []string{"sync"},
	}, h.Sync)

	huma.Register(api, huma.Operation{
		OperationID:   "webhook",
		Method:        http.MethodPost,
		Path:          "/api/v1/webhook/{source}",
		Summary:       "Accept a raw inbound webhook payload",
		Tags:          []string{"webhook"},
		DefaultStatus: http.StatusAccepted,
	}, h.Webhook)

	huma.Register(api, huma.Operation{
		OperationID: "magic-login",
		Method:      http.MethodPost,
		Path:        "/api/v1/magic-login",
		Summary:     "Request a passwordless magic-login token",
		Tags:        []string{"auth"},
	}, h.MagicLogin)
}
