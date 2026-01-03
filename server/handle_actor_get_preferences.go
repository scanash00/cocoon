package server

import (
	"encoding/json"

	"github.com/labstack/echo/v4"
)

// This is kinda lame. Not great to implement app.bsky in the pds, but alas

func (s *Server) handleActorGetPreferences(e echo.Context) error {
	repo, ok := getRepoFromContext(e)
	if !ok {
		return echo.NewHTTPError(401, "Unauthorized")
	}

	var prefs map[string]any
	err := json.Unmarshal(repo.Preferences, &prefs)
	if err != nil || prefs["preferences"] == nil {
		prefs = map[string]any{
			"preferences": []any{},
		}
	}

	return e.JSON(200, prefs)
}
