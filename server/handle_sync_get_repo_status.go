package server

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/labstack/echo/v4"
)

type ComAtprotoSyncGetRepoStatusResponse struct {
	Did    string  `json:"did"`
	Active bool    `json:"active"`
	Status *string `json:"status,omitempty"`
	Rev    *string `json:"rev,omitempty"`
}

func (s *Server) handleSyncGetRepoStatus(e echo.Context) error {
	ctx := e.Request().Context()

	did := e.QueryParam("did")
	if did == "" {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	urepo, err := s.getRepoActorByDid(ctx, did)
	if err != nil {
		return helpers.InputError(e, to.StringPtr("RepoNotFound"))
	}

	return e.JSON(200, ComAtprotoSyncGetRepoStatusResponse{
		Did:    urepo.Repo.Did,
		Active: urepo.Active(),
		Status: urepo.Status(),
		Rev:    &urepo.Rev,
	})
}
