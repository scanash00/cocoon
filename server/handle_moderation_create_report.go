package server

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/models"
	"github.com/labstack/echo/v4"
	secp256k1secec "gitlab.com/yawning/secp256k1-voi/secec"
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

	endpoint, svcDid, err := s.getAtprotoProxyEndpointFromRequest(e)
	if err != nil {
		s.logger.Error("could not get atproto proxy", "error", err)
		return helpers.ServerError(e, nil)
	}

	requrl := e.Request().URL
	requrl.Host = strings.TrimPrefix(endpoint, "https://")
	requrl.Scheme = "https"

	upReq, err := http.NewRequest(e.Request().Method, requrl.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	upReq.Header = e.Request().Header.Clone()

	header := map[string]string{
		"alg": "ES256K",
		"crv": "secp256k1",
		"typ": "JWT",
	}
	hj, err := json.Marshal(header)
	if err != nil {
		s.logger.Error("error marshaling header", "error", err)
		return helpers.ServerError(e, nil)
	}
	encheader := strings.TrimRight(base64.RawURLEncoding.EncodeToString(hj), "=")

	pts := strings.Split(e.Request().URL.Path, "/")
	if len(pts) != 3 {
		return fmt.Errorf("incorrect number of parts")
	}

	payload := map[string]any{
		"iss": repo.Repo.Did,
		"aud": svcDid,
		"lxm": pts[2],
		"jti": uuid.NewString(),
		"exp": time.Now().Add(1 * time.Minute).UTC().Unix(),
	}
	pj, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("error marashaling payload", "error", err)
		return helpers.ServerError(e, nil)
	}
	encpayload := strings.TrimRight(base64.RawURLEncoding.EncodeToString(pj), "=")

	input := fmt.Sprintf("%s.%s", encheader, encpayload)
	hash := sha256.Sum256([]byte(input))

	sk, err := secp256k1secec.NewPrivateKey(repo.SigningKey)
	if err != nil {
		s.logger.Error("can't load private key", "error", err)
		return err
	}

	R, S, _, err := sk.SignRaw(rand.Reader, hash[:])
	if err != nil {
		s.logger.Error("error signing", "error", err)
		return err
	}

	rBytes := R.Bytes()
	sBytes := S.Bytes()

	rPadded := make([]byte, 32)
	sPadded := make([]byte, 32)
	copy(rPadded[32-len(rBytes):], rBytes)
	copy(sPadded[32-len(sBytes):], sBytes)

	rawsig := append(rPadded, sPadded...)
	encsig := strings.TrimRight(base64.RawURLEncoding.EncodeToString(rawsig), "=")
	token := fmt.Sprintf("%s.%s", input, encsig)

	upReq.Header.Set("authorization", "Bearer "+token)

	upResp, err := http.DefaultClient.Do(upReq)
	if err != nil {
		return err
	}
	defer upResp.Body.Close()

	respBody, err := io.ReadAll(upResp.Body)
	if err != nil {
		s.logger.Error("error reading upstream response", "error", err)
		return helpers.ServerError(e, nil)
	}

	if upResp.StatusCode >= 200 && upResp.StatusCode < 300 {
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
	}

	for k, v := range upResp.Header {
		e.Response().Header().Set(k, strings.Join(v, ","))
	}

	contentType := upResp.Header.Get("content-type")
	return e.Blob(upResp.StatusCode, contentType, respBody)
}
