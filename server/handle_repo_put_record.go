package server

import (
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/labstack/echo/v4"
)

type ComAtprotoRepoPutRecordInput struct {
	Repo       string         `json:"repo" validate:"required,atproto-did"`
	Collection string         `json:"collection" validate:"required,atproto-nsid"`
	Rkey       string         `json:"rkey" validate:"required,atproto-rkey"`
	Validate   *bool          `json:"bool,omitempty"`
	Record     MarshalableMap `json:"record" validate:"required"`
	SwapRecord *string        `json:"swapRecord"`
	SwapCommit *string        `json:"swapCommit"`
}

func (s *Server) handlePutRecord(e echo.Context) error {
	ctx := e.Request().Context()

	repo, ok := getRepoFromContext(e)
	if !ok {
		return echo.NewHTTPError(401, "Unauthorized")
	}

	var req ComAtprotoRepoPutRecordInput
	if err := e.Bind(&req); err != nil {
		s.logger.Error("error binding", "error", err)
		return helpers.ServerError(e, nil)
	}

	if err := e.Validate(req); err != nil {
		s.logger.Error("error validating", "error", err)
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	if repo.Repo.Did != req.Repo {
		s.logger.Warn("mismatched repo/auth")
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	optype := OpTypeCreate
	if req.SwapRecord != nil {
		optype = OpTypeUpdate
	}

	results, err := s.repoman.applyWrites(ctx, repo.Repo, []Op{
		{
			Type:       optype,
			Collection: req.Collection,
			Rkey:       &req.Rkey,
			Validate:   req.Validate,
			Record:     &req.Record,
			SwapRecord: req.SwapRecord,
		},
	}, req.SwapCommit)
	if err != nil {
		s.logger.Error("error applying writes", "error", err)
		return helpers.ServerError(e, nil)
	}

	results[0].Type = nil

	return e.JSON(200, results[0])
}
