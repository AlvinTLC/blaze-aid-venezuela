package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/auth"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/email"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// userContextKey scopes the authenticated subject stored on the request context.
type userContextKey struct{}

// bearerSecurity marks an operation as requiring a valid bearer JWT.
var bearerSecurity = []map[string][]string{{"bearerAuth": {}}}

// WebhookEnqueuer persists a raw webhook payload and queues it for async
// processing, returning the new webhooks_log id.
type WebhookEnqueuer interface {
	EnqueueWebhook(ctx context.Context, source string, payload []byte) (string, error)
}

// Handler holds dependencies shared by every operation.
type Handler struct {
	repo       *repository.Repository
	enqueuer   WebhookEnqueuer
	email      email.EmailSender
	baseURL    string
	jwtSecret  string
	production bool
	logger     *slog.Logger
	api        huma.API
}

// New constructs a Handler. When production is true, sensitive stub behaviour
// (e.g. returning the magic-login token in the response) is disabled. baseURL is
// prefixed to magic links in outgoing email (empty = relative link).
func New(repo *repository.Repository, enqueuer WebhookEnqueuer, sender email.EmailSender, baseURL, jwtSecret string, production bool, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, enqueuer: enqueuer, email: sender, baseURL: baseURL, jwtSecret: jwtSecret, production: production, logger: logger}
}

// Register wires every P0 operation onto the Huma API and installs the JWT
// middleware that guards operations declaring bearerSecurity.
func (h *Handler) Register(api huma.API) {
	h.api = api
	api.UseMiddleware(h.authMiddleware)

	// Writes require a session JWT.
	huma.Register(api, huma.Operation{
		OperationID: "ingest-project",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/project",
		Summary:     "Ingest an aid project",
		Tags:        []string{"ingest"},
		Security:    bearerSecurity,
	}, h.IngestProject)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-resource",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/resource",
		Summary:     "Ingest a resource",
		Tags:        []string{"ingest"},
		Security:    bearerSecurity,
	}, h.IngestResource)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-missing",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/missing",
		Summary:     "Ingest a missing-person report",
		Tags:        []string{"ingest"},
		Security:    bearerSecurity,
	}, h.IngestMissing)

	huma.Register(api, huma.Operation{
		OperationID: "ingest-volunteer",
		Method:      http.MethodPost,
		Path:        "/api/v1/ingest/volunteer",
		Summary:     "Ingest a volunteer",
		Tags:        []string{"ingest"},
		Security:    bearerSecurity,
	}, h.IngestVolunteer)

	// Public read.
	huma.Register(api, huma.Operation{
		OperationID: "sync",
		Method:      http.MethodGet,
		Path:        "/api/v1/sync",
		Summary:     "Pull entity changes since a cursor",
		Tags:        []string{"sync"},
	}, h.Sync)

	// Public catalog reads (list + detail) for the frontend.
	huma.Register(api, huma.Operation{
		OperationID: "list-projects", Method: http.MethodGet, Path: "/api/v1/projects",
		Summary: "List aid projects", Tags: []string{"projects"},
	}, h.ListProjects)
	huma.Register(api, huma.Operation{
		OperationID: "get-project", Method: http.MethodGet, Path: "/api/v1/projects/{id}",
		Summary: "Get an aid project by id", Tags: []string{"projects"},
	}, h.GetProject)

	huma.Register(api, huma.Operation{
		OperationID: "list-resources", Method: http.MethodGet, Path: "/api/v1/resources",
		Summary: "List resources", Tags: []string{"resources"},
	}, h.ListResources)
	huma.Register(api, huma.Operation{
		OperationID: "get-resource", Method: http.MethodGet, Path: "/api/v1/resources/{id}",
		Summary: "Get a resource by id", Tags: []string{"resources"},
	}, h.GetResource)

	huma.Register(api, huma.Operation{
		OperationID: "list-missing", Method: http.MethodGet, Path: "/api/v1/missing",
		Summary: "List missing-person reports", Tags: []string{"missing"},
	}, h.ListMissing)
	huma.Register(api, huma.Operation{
		OperationID: "get-missing", Method: http.MethodGet, Path: "/api/v1/missing/{id}",
		Summary: "Get a missing-person report by id", Tags: []string{"missing"},
	}, h.GetMissing)

	huma.Register(api, huma.Operation{
		OperationID: "list-volunteers", Method: http.MethodGet, Path: "/api/v1/volunteers",
		Summary: "List volunteers", Tags: []string{"volunteers"},
	}, h.ListVolunteers)
	huma.Register(api, huma.Operation{
		OperationID: "get-volunteer", Method: http.MethodGet, Path: "/api/v1/volunteers/{id}",
		Summary: "Get a volunteer by id", Tags: []string{"volunteers"},
	}, h.GetVolunteer)

	// Mutations (Bearer): partial update + soft delete.
	huma.Register(api, huma.Operation{
		OperationID: "patch-project", Method: http.MethodPatch, Path: "/api/v1/projects/{id}",
		Summary: "Update an aid project", Tags: []string{"projects"}, Security: bearerSecurity,
	}, h.PatchProject)
	huma.Register(api, huma.Operation{
		OperationID: "delete-project", Method: http.MethodDelete, Path: "/api/v1/projects/{id}",
		Summary: "Soft-delete an aid project", Tags: []string{"projects"}, Security: bearerSecurity,
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteProject)

	huma.Register(api, huma.Operation{
		OperationID: "patch-resource", Method: http.MethodPatch, Path: "/api/v1/resources/{id}",
		Summary: "Update a resource", Tags: []string{"resources"}, Security: bearerSecurity,
	}, h.PatchResource)
	huma.Register(api, huma.Operation{
		OperationID: "delete-resource", Method: http.MethodDelete, Path: "/api/v1/resources/{id}",
		Summary: "Soft-delete a resource", Tags: []string{"resources"}, Security: bearerSecurity,
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteResource)

	huma.Register(api, huma.Operation{
		OperationID: "patch-missing", Method: http.MethodPatch, Path: "/api/v1/missing/{id}",
		Summary: "Update a missing-person report", Tags: []string{"missing"}, Security: bearerSecurity,
	}, h.PatchMissing)
	huma.Register(api, huma.Operation{
		OperationID: "delete-missing", Method: http.MethodDelete, Path: "/api/v1/missing/{id}",
		Summary: "Soft-delete a missing-person report", Tags: []string{"missing"}, Security: bearerSecurity,
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteMissing)

	huma.Register(api, huma.Operation{
		OperationID: "patch-volunteer", Method: http.MethodPatch, Path: "/api/v1/volunteers/{id}",
		Summary: "Update a volunteer", Tags: []string{"volunteers"}, Security: bearerSecurity,
	}, h.PatchVolunteer)
	huma.Register(api, huma.Operation{
		OperationID: "delete-volunteer", Method: http.MethodDelete, Path: "/api/v1/volunteers/{id}",
		Summary: "Soft-delete a volunteer", Tags: []string{"volunteers"}, Security: bearerSecurity,
		DefaultStatus: http.StatusNoContent,
	}, h.DeleteVolunteer)

	// Public ingestion from external systems (source auth handled separately).
	huma.Register(api, huma.Operation{
		OperationID:   "webhook",
		Method:        http.MethodPost,
		Path:          "/api/v1/webhook/{source}",
		Summary:       "Accept a raw inbound webhook payload",
		Tags:          []string{"webhook"},
		DefaultStatus: http.StatusAccepted,
	}, h.Webhook)

	// Auth bootstrap (public).
	huma.Register(api, huma.Operation{
		OperationID: "magic-login",
		Method:      http.MethodPost,
		Path:        "/api/v1/magic-login",
		Summary:     "Request a passwordless magic-login token",
		Tags:        []string{"auth"},
	}, h.MagicLogin)

	huma.Register(api, huma.Operation{
		OperationID: "auth-verify",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/verify",
		Summary:     "Exchange a magic token for a session JWT",
		Tags:        []string{"auth"},
	}, h.AuthVerify)
}

// authMiddleware enforces a valid bearer JWT on operations that declare security,
// and is a no-op for public operations.
func (h *Handler) authMiddleware(ctx huma.Context, next func(huma.Context)) {
	if len(ctx.Operation().Security) == 0 {
		next(ctx)
		return
	}

	const prefix = "Bearer "
	authz := ctx.Header("Authorization")
	if !strings.HasPrefix(authz, prefix) {
		_ = huma.WriteErr(h.api, ctx, http.StatusUnauthorized, "missing bearer token")
		return
	}

	subject, err := auth.ParseJWT(h.jwtSecret, strings.TrimPrefix(authz, prefix))
	if err != nil {
		_ = huma.WriteErr(h.api, ctx, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	next(huma.WithValue(ctx, userContextKey{}, subject))
}
