package server

import (
	"strconv"
	
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/ipfs/go-cid"
	"github.com/labstack/echo/v4"
)

type ComAtprotoSyncListReposResponse struct {
	Cursor *string                           `json:"cursor,omitempty"`
	Repos  []ComAtprotoSyncListReposRepoItem `json:"repos"`
}

type ComAtprotoSyncListReposRepoItem struct {
	Did    string  `json:"did"`
	Head   string  `json:"head"`
	Rev    string  `json:"rev"`
	Active bool    `json:"active"`
	Status *string `json:"status,omitempty"`
}

func (s *Server) handleListRepos(e echo.Context) error {
	ctx := e.Request().Context()

	limit := 500
	if limitStr := e.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	cursor := e.QueryParam("cursor")
	query := "SELECT * FROM repos"
	args := []any{}

	if cursor != "" {
		query += " WHERE did > ?"
		args = append(args, cursor)
	}

	query += " ORDER BY did ASC LIMIT ?"
	args = append(args, limit)

	var repos []models.Repo
	if err := s.db.Raw(ctx, query, nil, args...).Scan(&repos).Error; err != nil {
		s.logger.Error("error listing repos", "error", err)
		return helpers.ServerError(e, nil)
	}

	var items []ComAtprotoSyncListReposRepoItem
	for _, r := range repos {
		c, err := cid.Cast(r.Root)
		if err != nil {
			s.logger.Error("error casting root cid", "error", err)
			continue
		}

		items = append(items, ComAtprotoSyncListReposRepoItem{
			Did:    r.Did,
			Head:   c.String(),
			Rev:    r.Rev,
			Active: r.Active(),
			Status: r.Status(),
		})
	}

	var newCursor *string
	if len(repos) == limit {
		newCursor = to.StringPtr(repos[len(repos)-1].Did)
	}

	return e.JSON(200, ComAtprotoSyncListReposResponse{
		Cursor: newCursor,
		Repos:  items,
	})
}
