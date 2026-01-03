package server

import (
	"encoding/json"
	
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/labstack/echo/v4"
)

// This is kinda lame. Not great to implement app.bsky in the pds, but alas

func (s *Server) handleActorPutPreferences(e echo.Context) error {
	ctx := e.Request().Context()

	repo, ok := getRepoFromContext(e)
	if !ok {
		return echo.NewHTTPError(401, "Unauthorized")
	}

	var prefs map[string]any
	if err := json.NewDecoder(e.Request().Body).Decode(&prefs); err != nil {
		s.logger.Error("error decoding preferences", "error", err); return helpers.ServerError(e, nil)
	}

	b, err := json.Marshal(prefs)
	if err != nil {
		s.logger.Error("error marshaling preferences", "error", err); return helpers.ServerError(e, nil)
	}

	if err := s.db.Exec(ctx, "UPDATE repos SET preferences = ? WHERE did = ?", nil, b, repo.Repo.Did).Error; err != nil {
		s.logger.Error("error saving preferences to db", "error", err); return helpers.ServerError(e, nil)
	}

	return e.NoContent(200)
}
