package server

import (
	"context"
	"io"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/repo"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	blocks "github.com/ipfs/go-block-format"
	"github.com/ipfs/go-cid"
	"github.com/ipld/go-car"
	"github.com/labstack/echo/v4"
)

const importBatchSize = 100

func (s *Server) handleRepoImportRepo(e echo.Context) error {
	ctx := e.Request().Context()
	logger := s.logger.With("name", "handleImportRepo")

	urepo := e.Get("repo").(*models.RepoActor)
	bs := s.getBlockstore(urepo.Repo.Did)

	cs, err := car.NewCarReader(e.Request().Body)
	if err != nil {
		logger.Error("could not read car in import request", "error", err)
		return helpers.ServerError(e, nil)
	}

	var batch []blocks.Block
	blockCount := 0

	for {
		block, err := cs.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("error reading block from car", "error", err, "block", blockCount)
			return helpers.ServerError(e, nil)
		}

		batch = append(batch, block)
		blockCount++

		if len(batch) >= importBatchSize {
			if err := bs.PutMany(context.TODO(), batch); err != nil {
				logger.Error("could not insert blocks batch", "error", err)
				return helpers.ServerError(e, nil)
			}
			logger.Debug("imported block batch", "count", blockCount)
			batch = batch[:0]
		}
	}

	if len(batch) > 0 {
		if err := bs.PutMany(context.TODO(), batch); err != nil {
			logger.Error("could not insert remaining blocks", "error", err)
			return helpers.ServerError(e, nil)
		}
	}

	logger.Info("imported repo blocks", "total", blockCount, "did", urepo.Repo.Did)

	r, err := repo.OpenRepo(context.TODO(), bs, cs.Header.Roots[0])
	if err != nil {
		logger.Error("could not open repo", "error", err)
		return helpers.ServerError(e, nil)
	}

	tx := s.db.BeginDangerously(ctx)

	clock := syntax.NewTIDClock(0)

	if err := r.ForEach(context.TODO(), "", func(key string, cid cid.Cid) error {
		pts := strings.Split(key, "/")
		nsid := pts[0]
		rkey := pts[1]
		cidStr := cid.String()
		b, err := bs.Get(context.TODO(), cid)
		if err != nil {
			logger.Error("record bytes don't exist in blockstore", "error", err)
			return helpers.ServerError(e, nil)
		}

		rec := models.Record{
			Did:       urepo.Repo.Did,
			CreatedAt: clock.Next().String(),
			Nsid:      nsid,
			Rkey:      rkey,
			Cid:       cidStr,
			Value:     b.RawData(),
		}

		if err := tx.Save(rec).Error; err != nil {
			return err
		}

		return nil
	}); err != nil {
		tx.Rollback()
		logger.Error("record bytes don't exist in blockstore", "error", err)
		return helpers.ServerError(e, nil)
	}

	tx.Commit()

	root, rev, err := r.Commit(context.TODO(), urepo.SignFor)
	if err != nil {
		logger.Error("error committing", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := s.UpdateRepo(context.TODO(), urepo.Repo.Did, root, rev); err != nil {
		logger.Error("error updating repo after commit", "error", err)
		return helpers.ServerError(e, nil)
	}

	return nil
}
