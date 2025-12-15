package server

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
)

type ComAtprotoModerationCreateReportRequest struct {
	ReasonType string          `json:"reasonType"`
	Reason     *string         `json:"reason,omitempty"`
	Subject    json.RawMessage `json:"subject"`
}

func (s *Server) handleModerationCreateReport(e echo.Context) error {
	repo, ok := e.Get("repo").(*models.RepoActor)
	if !ok || repo == nil {
		return helpers.AuthRequiredError(e, "Unauthorized", "Unauthorized")
	}

	bodyBytes, err := io.ReadAll(e.Request().Body)
	if err != nil {
		s.logger.Error("error reading createReport body", "error", err)
		return helpers.ServerError(e, nil)
	}
	_ = e.Request().Body.Close()
	e.Request().Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var req ComAtprotoModerationCreateReportRequest
	if err := json.Unmarshal(bodyBytes, &req); err == nil {
		appealText := ""
		if req.Reason != nil {
			appealText = *req.Reason
		}

		subjectJSON := ""
		if len(req.Subject) > 0 {
			subjectJSON = string(req.Subject)
		}

		if err := s.sendAppealNotice(repo.Repo.Did, repo.Handle, repo.Repo.TakedownComment, req.ReasonType, appealText, subjectJSON); err != nil {
			s.logger.Error("error sending appeal notice email", "error", err)
			return helpers.ServerError(e, nil)
		}
	} else {
		s.logger.Error("error parsing createReport body", "error", err)
	}

	return s.handleProxy(e)
}
