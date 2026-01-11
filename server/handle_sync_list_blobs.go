package server

import (
	"strconv"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/ipfs/go-cid"
	"github.com/labstack/echo/v4"
)

type ComAtprotoSyncListBlobsResponse struct {
	Cursor *string  `json:"cursor,omitempty"`
	Cids   []string `json:"cids"`
}

func (s *Server) handleSyncListBlobs(e echo.Context) error {
	ctx := e.Request().Context()

	did := e.QueryParam("did")
	if did == "" {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	// cursor is time-based
	cursor := e.QueryParam("cursor")

	limit := 500
	if limitStr := e.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	cursorquery := ""

	params := []any{did}
	if cursor != "" {
		params = append(params, cursor)
		cursorquery = "AND created_at < ?"
	}
	params = append(params, limit)

	urepo, err := s.getRepoActorByDid(ctx, did)
	if err != nil {
		s.logger.Error("could not find user for requested blobs", "error", err)
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	status := urepo.Status()
	if status != nil {
		if *status == "deactivated" {
			return helpers.InputError(e, to.StringPtr("RepoDeactivated"))
		}
	}

	var blobs []models.Blob
	if err := s.db.Raw(ctx, "SELECT * FROM blobs WHERE did = ? "+cursorquery+" ORDER BY created_at DESC LIMIT ?", nil, params...).Scan(&blobs).Error; err != nil {
		s.logger.Error("error getting records", "error", err)
		return helpers.ServerError(e, nil)
	}

	var cstrs []string
	for _, b := range blobs {
		c, err := cid.Cast(b.Cid)
		if err != nil {
			s.logger.Warn("skipping malformed cid in blob list", "error", err, "blob_id", b.ID, "did", b.Did)
			continue
		}
		cstrs = append(cstrs, c.String())
	}

	var newcursor *string
	if len(blobs) == limit {
		newcursor = &blobs[len(blobs)-1].CreatedAt
	}

	return e.JSON(200, ComAtprotoSyncListBlobsResponse{
		Cursor: newcursor,
		Cids:   cstrs,
	})
}
