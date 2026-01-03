package server

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
)

type ComAtprotoServerUpdateEmailAuthFactorRequest struct {
	EmailAuthFactorEnabled bool `json:"emailAuthFactor" validate:""`
}

func (s *Server) handleUpdateEmailAuthFactor(e echo.Context) error {
	ctx := e.Request().Context()
	logger := s.logger.With("name", "handleUpdateEmailAuthFactor")

	repo := e.Get("repo").(*models.RepoActor)

	var req ComAtprotoServerUpdateEmailAuthFactorRequest
	if err := e.Bind(&req); err != nil {
		logger.Error("error binding request", "error", err)
		return helpers.ServerError(e, nil)
	}

	if req.EmailAuthFactorEnabled && repo.EmailConfirmedAt == nil {
		return helpers.InputError(e, to.StringPtr("EmailNotConfirmed"))
	}

	if err := s.db.Exec(ctx, "UPDATE repos SET email_auth_factor_enabled = ? WHERE did = ?",
		nil, req.EmailAuthFactorEnabled, repo.Repo.Did).Error; err != nil {
		logger.Error("error updating email auth factor", "error", err)
		return helpers.ServerError(e, nil)
	}

	return e.NoContent(200)
}
