package handler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/auth"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
)

// userContextKey scopes the authenticated subject stored on the request context.
type userContextKey struct{}

// bearerSecurity marks an operation as requiring a valid bearer JWT.
var bearerSecurity = []map[string][]string{{"bearerAuth": {}}}

// Handler holds dependencies shared by every operation.
type Handler struct {
	repo       *repository.Repository
	jwtSecret  string
	production bool
	logger     *slog.Logger
	api        huma.API
}

// New constructs a Handler. When production is true, sensitive stub behaviour
// (e.g. returning the magic-login token in the response) is disabled.
func New(repo *repository.Repository, jwtSecret string, production bool, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, jwtSecret: jwtSecret, production: production, logger: logger}
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
