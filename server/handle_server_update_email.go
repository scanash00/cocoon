package server

import (
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
)

type ComAtprotoServerUpdateEmailRequest struct {
	Email           string `json:"email" validate:"required"`
	EmailAuthFactor bool   `json:"emailAuthFactor"`
	Token           string `json:"token"` // Optional
}

func (s *Server) handleServerUpdateEmail(e echo.Context) error {
	ctx := e.Request().Context()
	logger := s.logger.With("name", "handleServerUpdateEmail")

	urepo := e.Get("repo").(*models.RepoActor)

	var req ComAtprotoServerUpdateEmailRequest
	if err := e.Bind(&req); err != nil {
		logger.Error("error binding", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := e.Validate(req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	tokenRequired := false

	if req.Email != urepo.Email && urepo.EmailConfirmedAt != nil {
		tokenRequired = true
	}

	if req.EmailAuthFactor != urepo.EmailAuthFactorEnabled {
		tokenRequired = true
	}
	if req.EmailAuthFactor && !urepo.EmailAuthFactorEnabled {
		if urepo.EmailConfirmedAt == nil && req.Email == urepo.Email {
			return helpers.InputError(e, to.StringPtr("EmailNotConfirmed"))
		}
	}

	if tokenRequired || req.Token != "" {
		if req.Token == "" {
			return helpers.InputError(e, to.StringPtr("TokenRequired"))
		}

		if urepo.EmailUpdateCode == nil || urepo.EmailUpdateCodeExpiresAt == nil {
			return helpers.InvalidTokenError(e)
		}

		cleanToken := strings.TrimSpace(req.Token)
		if !strings.EqualFold(*urepo.EmailUpdateCode, cleanToken) {
			return helpers.InvalidTokenError(e)
		}

		if time.Now().UTC().After(*urepo.EmailUpdateCodeExpiresAt) {
			return helpers.ExpiredTokenError(e)
		}

	}

	newEmail := req.Email
	newAuthFactor := req.EmailAuthFactor
	var newConfirmedAt *time.Time = urepo.EmailConfirmedAt

	if req.Email != urepo.Email {
		newConfirmedAt = nil
		newAuthFactor = false
	}

	query := "UPDATE repos SET email = ?, email_auth_factor_enabled = ?, email_confirmed_at = ?"
	args := []interface{}{newEmail, newAuthFactor, newConfirmedAt}

	if tokenRequired || req.Token != "" {
		query += ", email_update_code = NULL, email_update_code_expires_at = NULL"
	}

	query += " WHERE did = ?"
	args = append(args, urepo.Repo.Did)

	if err := s.db.Exec(ctx, query, nil, args...).Error; err != nil {
		logger.Error("error updating repo", "error", err)
		return helpers.ServerError(e, nil)
	}

	return e.NoContent(200)
}
