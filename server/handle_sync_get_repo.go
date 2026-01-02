package server

import (


	"github.com/bluesky-social/indigo/carstore"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/ipld/go-car"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleSyncGetRepo(e echo.Context) error {
	ctx := e.Request().Context()

	did := e.QueryParam("did")
	if did == "" {
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	urepo, err := s.getRepoActorByDid(ctx, did)
	if err != nil {
		return helpers.InputError(e, to.StringPtr("RepoNotFound"))
	}

	rc, err := cid.Cast(urepo.Root)
	if err != nil {
		s.logger.Error("error casting root cid", "error", err)
		return helpers.ServerError(e, nil)
	}

	hb, err := cbor.DumpObject(&car.CarHeader{
		Roots:   []cid.Cid{rc},
		Version: 1,
	})

	e.Response().Header().Set(echo.HeaderContentType, "application/vnd.ipld.car")
	e.Response().WriteHeader(200)

	if _, err := carstore.LdWrite(e.Response().Writer, hb); err != nil {
		s.logger.Error("error writing to car", "error", err)
		return nil
	}

	rows, err := s.db.Raw(ctx, "SELECT cid, value FROM blocks WHERE did = ? ORDER BY rev ASC", nil, urepo.Repo.Did).Rows()
	if err != nil {
		s.logger.Error("error getting blocks", "error", err)
		return nil
	}
	defer rows.Close()

	var bCid []byte
	var bVal []byte
	for rows.Next() {
		if err := rows.Scan(&bCid, &bVal); err != nil {
			s.logger.Error("error scanning block", "error", err)
			continue
		}
		
		c, err := cid.Cast(bCid)
		if err != nil {
			continue
		}

		if _, err := carstore.LdWrite(e.Response().Writer, c.Bytes(), bVal); err != nil {
			s.logger.Error("error writing block to car", "error", err)
			return nil
		}
	}

	return nil
}
