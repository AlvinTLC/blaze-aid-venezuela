package handler

import (
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

// init overrides Huma's default RFC7807 error model with a simple
// {"error":{"code","message"}} envelope (success list responses keep their
// {items,...} shape). Runs on import so the app and tests share it.
func init() {
	huma.NewError = func(status int, message string, errs ...error) huma.StatusError {
		msg := message
		if details := joinErrs(errs); details != "" {
			msg = message + ": " + details
		}
		return &ErrorEnvelope{Status: status, Err: ErrorBody{Code: status, Message: msg}}
	}
}

// ErrorBody is the inner error payload.
type ErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorEnvelope is the wire shape for all error responses.
type ErrorEnvelope struct {
	Status int       `json:"-"`
	Err    ErrorBody `json:"error"`
}

func (e *ErrorEnvelope) Error() string  { return e.Err.Message }
func (e *ErrorEnvelope) GetStatus() int { return e.Status }

func joinErrs(errs []error) string {
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		if e != nil {
			parts = append(parts, e.Error())
		}
	}
	return strings.Join(parts, "; ")
}
