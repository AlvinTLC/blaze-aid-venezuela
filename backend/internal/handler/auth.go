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
// NOTE: for P0/beta we return the token directly; production must instead
// email the link and never expose the token in the response body.
type MagicLoginOutput struct {
	Body struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
		MagicLink string    `json:"magic_link"`
	}
}

// MagicLogin mints a single-use token bound to the supplied email.
func (h *Handler) MagicLogin(ctx context.Context, in *MagicLoginInput) (*MagicLoginOutput, error) {
	token, expiresAt, err := h.repo.CreateMagicToken(ctx, in.Body.Email, magicTokenTTL)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to issue magic token", err)
	}

	out := &MagicLoginOutput{}
	out.Body.Token = token
	out.Body.ExpiresAt = expiresAt
	out.Body.MagicLink = "/auth/verify?token=" + token
	return out, nil
}
