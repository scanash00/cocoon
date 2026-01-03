package server

import (
	"context"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/events"
	"github.com/bluesky-social/indigo/util"
	"github.com/haileyok/cocoon/identity"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/plc"
	"github.com/labstack/echo/v4"
)

type ComAtprotoIdentityUpdateHandleRequest struct {
	Handle string `json:"handle" validate:"atproto-handle"`
}

func (s *Server) handleIdentityUpdateHandle(e echo.Context) error {
	logger := s.logger.With("name", "handleIdentityUpdateHandle")

	repo, ok := getRepoFromContext(e)
	if !ok {
		return echo.NewHTTPError(401, "Unauthorized")
	}

	var req ComAtprotoIdentityUpdateHandleRequest
	if err := e.Bind(&req); err != nil {
		logger.Error("error binding", "error", err)
		return helpers.ServerError(e, nil)
	}

	req.Handle = strings.ToLower(req.Handle)

	if err := e.Validate(req); err != nil {
		return helpers.InputError(e, to.StringPtr("InvalidHandle"))
	}

	ctx := context.WithValue(e.Request().Context(), "skip-cache", true)

	if strings.HasPrefix(repo.Repo.Did, "did:plc:") {
		log, err := identity.FetchDidAuditLog(ctx, nil, repo.Repo.Did)
		if err != nil {
			logger.Error("error fetching doc", "error", err)
			return helpers.ServerError(e, nil)
		}

		latest := log[len(log)-1]

		var newAka []string
		for _, aka := range latest.Operation.AlsoKnownAs {
			if aka == "at://"+repo.Handle {
				continue
			}
			newAka = append(newAka, aka)
		}

		newAka = append(newAka, "at://"+req.Handle)

		op := plc.Operation{
			Type:                "plc_operation",
			VerificationMethods: latest.Operation.VerificationMethods,
			RotationKeys:        latest.Operation.RotationKeys,
			AlsoKnownAs:         newAka,
			Services:            latest.Operation.Services,
			Prev:                &latest.Cid,
		}

		k, err := atcrypto.ParsePrivateBytesK256(repo.SigningKey)
		if err != nil {
			logger.Error("error parsing signing key", "error", err)
			return helpers.ServerError(e, nil)
		}

		if err := s.plcClient.SignOp(k, &op); err != nil {
			s.logger.Error("error signing plc operation", "error", err)
			return helpers.ServerError(e, nil)
		}

		if err := s.plcClient.SendOperation(e.Request().Context(), repo.Repo.Did, &op); err != nil {
			s.logger.Error("error sending plc operation", "error", err)
			return helpers.ServerError(e, nil)
		}
	}

	if err := s.passport.BustDoc(ctx, repo.Repo.Did); err != nil {
		logger.Warn("error busting did doc", "error", err)
	}

	s.evtman.AddEvent(ctx, &events.XRPCStreamEvent{
		RepoIdentity: &atproto.SyncSubscribeRepos_Identity{
			Did:    repo.Repo.Did,
			Handle: to.StringPtr(req.Handle),
			Seq:    s.nextSeq(ctx),
			Time:   time.Now().Format(util.ISO8601),
		},
	})

	if err := s.db.Exec(ctx, "UPDATE actors SET handle = ? WHERE did = ?", nil, req.Handle, repo.Repo.Did).Error; err != nil {
		logger.Error("error updating handle in db", "error", err)
		return helpers.ServerError(e, nil)
	}

	return e.NoContent(200)
}
