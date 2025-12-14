package server

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
)

type ComAtprotoRepoGetRecordResponse struct {
	Uri   string         `json:"uri"`
	Cid   string         `json:"cid"`
	Value map[string]any `json:"value"`
}

func (s *Server) handleRepoGetRecord(e echo.Context) error {
	repo := e.QueryParam("repo")
	collection := e.QueryParam("collection")
	rkey := e.QueryParam("rkey")
	cidstr := e.QueryParam("cid")

	did := repo
	if _, err := syntax.ParseDID(did); err != nil {
		actor, err := s.getActorByHandle(repo)
		if err == nil {
			did = actor.Did
		}
	}

	urepo, err := s.getRepoActorByDid(did)
	if err == nil {
		status := urepo.Status()
		if status != nil {
			switch *status {
			case "takendown":
				return helpers.InputError(e, to.StringPtr("RepoTakendown"))
			case "deactivated":
				return helpers.InputError(e, to.StringPtr("RepoDeactivated"))
			}
		}
	}

	params := []any{did, collection, rkey}
	cidquery := ""

	if cidstr != "" {
		c, err := syntax.ParseCID(cidstr)
		if err != nil {
			return err
		}
		params = append(params, c.String())
		cidquery = " AND cid = ?"
	}

	var record models.Record
	if err := s.db.Raw("SELECT * FROM records WHERE did = ? AND nsid = ? AND rkey = ?"+cidquery, nil, params...).Scan(&record).Error; err != nil {
		// TODO: handle error nicely
		return err
	}

	val, err := atdata.UnmarshalCBOR(record.Value)
	if err != nil {
		return s.handleProxy(e) // TODO: this should be getting handled like...if we don't find it in the db. why doesn't it throw error up there?
	}

	return e.JSON(200, ComAtprotoRepoGetRecordResponse{
		Uri:   "at://" + record.Did + "/" + record.Nsid + "/" + record.Rkey,
		Cid:   record.Cid,
		Value: val,
	})
}
