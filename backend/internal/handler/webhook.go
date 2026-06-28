package handler

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
)

// WebhookInput accepts an arbitrary payload from a named source.
// RawBody is populated by Huma with the unparsed request body so we can
// persist heterogeneous third-party payloads verbatim for async processing.
type WebhookInput struct {
	Source  string `path:"source" doc:"Identifier of the upstream system" example:"telegram"`
	RawBody []byte
}

// WebhookOutput acknowledges receipt of the queued event.
type WebhookOutput struct {
	Body struct {
		ID     string `json:"id"`
		Status string `json:"status" example:"queued"`
	}
}

// Webhook stores the raw payload in raw_events for later (River) processing.
func (h *Handler) Webhook(ctx context.Context, in *WebhookInput) (*WebhookOutput, error) {
	payload := in.RawBody
	if len(payload) == 0 {
		payload = []byte("{}")
	}

	id, err := h.repo.InsertRawEvent(ctx, in.Source, payload)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to enqueue webhook", err)
	}

	out := &WebhookOutput{}
	out.Body.ID = id
	out.Body.Status = "queued"
	return out, nil
}
