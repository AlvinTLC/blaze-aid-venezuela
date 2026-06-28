package handler

import (
	"context"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/auth"
	"github.com/AlvinTLC/blaze-aid-venezuela/backend/internal/repository"
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

// MagicLogin mints a single-use token bound to the supplied email and emails the
// magic link. In non-production the token is also returned for testing.
func (h *Handler) MagicLogin(ctx context.Context, in *MagicLoginInput) (*MagicLoginOutput, error) {
	token, expiresAt, err := h.repo.CreateMagicToken(ctx, in.Body.Email, magicTokenTTL)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to issue magic token", err)
	}

	link := h.baseURL + "/auth/verify?token=" + token
	html, text := magicLinkEmail(link)
	if err := h.email.Send(ctx, in.Body.Email, "Tu acceso a BlazeAid Hub", html, text); err != nil {
		h.logger.Error("failed to send magic-login email", "err", err)
		if h.production {
			// In prod the email is the only delivery channel; surface the failure.
			return nil, huma.Error502BadGateway("could not send login email")
		}
	}

	out := &MagicLoginOutput{}
	out.Body.Status = "sent"
	if !h.production {
		// Dev/beta only: expose the token so clients can test the flow.
		out.Body.Token = token
		out.Body.ExpiresAt = expiresAt
		out.Body.MagicLink = link
	}
	return out, nil
}

// magicLinkEmail renders the HTML + plaintext bodies for the login email.
func magicLinkEmail(link string) (html, text string) {
	html = `<!doctype html><html><body style="font-family:sans-serif">
<h2>BlazeAid Hub</h2>
<p>Toca el botón para iniciar sesión. El enlace caduca en 15 minutos.</p>
<p><a href="` + link + `" style="display:inline-block;padding:12px 20px;background:#c1121f;color:#fff;text-decoration:none;border-radius:6px">Iniciar sesión</a></p>
<p>Si el botón no funciona, copia este enlace:<br>` + link + `</p>
</body></html>`
	text = "BlazeAid Hub\n\nInicia sesión con este enlace (caduca en 15 minutos):\n" + link + "\n"
	return html, text
}

// AuthVerifyInput carries the magic token to exchange for a session JWT.
type AuthVerifyInput struct {
	Body struct {
		Token string `json:"token" minLength:"1" doc:"Single-use magic token from magic-login"`
	}
}

// AuthVerifyOutput returns the issued bearer session token.
type AuthVerifyOutput struct {
	Body struct {
		AccessToken string    `json:"access_token"`
		TokenType   string    `json:"token_type" example:"Bearer"`
		ExpiresAt   time.Time `json:"expires_at"`
	}
}

// AuthVerify burns a valid magic token and issues a signed session JWT.
func (h *Handler) AuthVerify(ctx context.Context, in *AuthVerifyInput) (*AuthVerifyOutput, error) {
	email, err := h.repo.ConsumeMagicToken(ctx, in.Body.Token)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidToken) {
			return nil, huma.Error401Unauthorized("invalid, used, or expired token")
		}
		return nil, huma.Error500InternalServerError("failed to verify token", err)
	}

	jwtStr, exp, err := auth.IssueJWT(h.jwtSecret, email)
	if err != nil {
		return nil, huma.Error500InternalServerError("failed to issue session token", err)
	}

	out := &AuthVerifyOutput{}
	out.Body.AccessToken = jwtStr
	out.Body.TokenType = "Bearer"
	out.Body.ExpiresAt = exp
	return out, nil
}
