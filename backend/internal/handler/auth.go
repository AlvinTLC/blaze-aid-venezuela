package handler

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

const magicTokenTTL = 15 * time.Minute

// MagicLoginInput requests a passwordless login token for an email.
type MagicLoginInput struct {
	Body struct {
		Email string `json:"email" format:"email" doc:"Email to receive the magic link"`
	}
}

// MagicLoginOutput returns the issued token.
//
// SECURITY: returning the token in the response body is a development/beta
// convenience that lets anyone who can POST an email mint a usable login token
// (account-takeover risk). It is therefore suppressed when ENV=production, where
// the token must instead be delivered out-of-band (email). Fields are omitempty
// so the production response carries only the acknowledgement.
type MagicLoginOutput struct {
	Body struct {
		Status    string    `json:"status"`
		Token     string    `json:"token,omitempty"`
		ExpiresAt time.Time `json:"expires_at,omitempty"`
		MagicLink string    `json:"magic_link,omitempty"`
	}
}

// MagicLogin mints a single-use token bound to the supplied email.
func (h *Handler) MagicLogin(ctx context.Context, in *MagicLoginInput) (*MagicLoginOutput, error) {
	token, expiresAt, err := h.repo.CreateMagicToken(ctx, in.Body.Email, magicTokenTTL)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to issue magic token", err)
	}

	out := &MagicLoginOutput{}
	out.Body.Status = "sent"
	if !h.production {
		// Dev/beta only: expose the token so clients can test the flow.
		out.Body.Token = token
		out.Body.ExpiresAt = expiresAt
		out.Body.MagicLink = "/auth/verify?token=" + token
	}
	return out, nil
}
