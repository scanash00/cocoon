package server

import (
	"context"
	"time"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/util"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/labstack/echo/v4"
)

type ComAtprotoServerTakedownAccountRequest struct {
	Did string `json:"did" validate:"required,atproto-did"`
}

func (s *Server) handleServerTakedownAccount(e echo.Context) error {
	var req ComAtprotoServerTakedownAccountRequest
	if err := e.Bind(&req); err != nil {
		s.logger.Error("error binding", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := e.Validate(&req); err != nil {
		s.logger.Error("error validating", "error", err)
		return helpers.InputError(e, nil)
	}

	if err := s.db.Exec("UPDATE repos SET takendown = ? WHERE did = ?", nil, true, req.Did).Error; err != nil {
		s.logger.Error("error updating account status to takendown", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := s.db.Exec("DELETE FROM tokens WHERE did = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting tokens", "error", err)
		return helpers.ServerError(e, nil)
	}
	if err := s.db.Exec("DELETE FROM refresh_tokens WHERE did = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting refresh tokens", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := s.db.Exec("DELETE FROM oauth_tokens WHERE sub = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting oauth tokens", "error", err)
		return helpers.ServerError(e, nil)
	}

	repo, err := s.getRepoActorByDid(req.Did)
	if err != nil {
		s.logger.Error("error fetching repo", "error", err)
		return helpers.ServerError(e, nil)
	}

	s.evtman.AddEvent(context.TODO(), &events.XRPCStreamEvent{
		RepoAccount: &atproto.SyncSubscribeRepos_Account{
			Active: repo.Active(),
			Did:    repo.Repo.Did,
			Status: repo.Status(),
			Seq:    time.Now().UnixMicro(),
			Time:   time.Now().Format(util.ISO8601),
		},
	})

	return e.NoContent(200)
}

type ComAtprotoServerUntakedownAccountRequest struct {
	Did string `json:"did" validate:"required,atproto-did"`
}

func (s *Server) handleServerUntakedownAccount(e echo.Context) error {
	var req ComAtprotoServerUntakedownAccountRequest
	if err := e.Bind(&req); err != nil {
		s.logger.Error("error binding", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := e.Validate(&req); err != nil {
		s.logger.Error("error validating", "error", err)
		return helpers.InputError(e, nil)
	}

	if err := s.db.Exec("UPDATE repos SET takendown = ? WHERE did = ?", nil, false, req.Did).Error; err != nil {
		s.logger.Error("error updating account status to untakedown", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := s.db.Exec("DELETE FROM tokens WHERE did = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting tokens", "error", err)
		return helpers.ServerError(e, nil)
	}
	if err := s.db.Exec("DELETE FROM refresh_tokens WHERE did = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting refresh tokens", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := s.db.Exec("DELETE FROM oauth_tokens WHERE sub = ?", nil, req.Did).Error; err != nil {
		s.logger.Error("error deleting oauth tokens", "error", err)
		return helpers.ServerError(e, nil)
	}

	repo, err := s.getRepoActorByDid(req.Did)
	if err != nil {
		s.logger.Error("error fetching repo", "error", err)
		return helpers.ServerError(e, nil)
	}

	s.evtman.AddEvent(context.TODO(), &events.XRPCStreamEvent{
		RepoAccount: &atproto.SyncSubscribeRepos_Account{
			Active: repo.Active(),
			Did:    repo.Repo.Did,
			Status: repo.Status(),
			Seq:    time.Now().UnixMicro(),
			Time:   time.Now().Format(util.ISO8601),
		},
	})

	return e.NoContent(200)
}
