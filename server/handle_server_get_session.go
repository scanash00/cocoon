package server

import (
	"github.com/labstack/echo/v4"
)

type ComAtprotoServerGetSessionResponse struct {
	Handle          string  `json:"handle"`
	Did             string  `json:"did"`
	Email           string  `json:"email"`
	EmailConfirmed  bool    `json:"emailConfirmed"`
	EmailAuthFactor bool    `json:"emailAuthFactor"`
	Active          bool    `json:"active"`
	Status          *string `json:"status,omitempty"`
}

func (s *Server) handleGetSession(e echo.Context) error {
	repo, ok := getRepoFromContext(e)
	if !ok {
		return echo.NewHTTPError(401, "Unauthorized")
	}

	return e.JSON(200, ComAtprotoServerGetSessionResponse{
		Handle:          repo.Handle,
		Did:             repo.Repo.Did,
		Email:           repo.Email,
		EmailConfirmed:  repo.EmailConfirmedAt != nil,
		EmailAuthFactor: repo.EmailAuthFactorEnabled,
		Active:          repo.Active(),
		Status:          repo.Status(),
	})
}
