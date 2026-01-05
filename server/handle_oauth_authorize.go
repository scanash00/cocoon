package server

import (
	"net/url"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/haileyok/cocoon/internal/helpers"
	"github.com/haileyok/cocoon/oauth"
	"github.com/haileyok/cocoon/oauth/constants"
	"github.com/haileyok/cocoon/oauth/provider"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleOauthAuthorizeGet(e echo.Context) error {
	ctx := e.Request().Context()

	var reqId string
	reqUri := e.QueryParam("request_uri")
	if reqUri != "" {
		id, err := oauth.DecodeRequestUri(reqUri)
		if err != nil {
			return helpers.InputError(e, to.StringPtr(err.Error()))
		}
		reqId = id
	} else {
		var parRequest provider.ParRequest
		if err := e.Bind(&parRequest); err != nil {
			s.logger.Error("error binding for standard auth request", "error", err)
			return helpers.InputError(e, to.StringPtr("InvalidRequest"))
		}

		if err := e.Validate(parRequest); err != nil {
			// render page for logged out dev
			if s.config.Version == "dev" && parRequest.ClientID == "" {
				return e.Render(200, "authorize.html", map[string]any{
					"Scopes":     []string{"atproto", "transition:generic"},
					"AppName":    "DEV MODE AUTHORIZATION PAGE",
					"Handle":     "paula.cocoon.social",
					"RequestUri": "",
				})
			}
			return helpers.InputError(e, to.StringPtr("no request uri and invalid parameters"))
		}

		client, clientAuth, err := s.oauthProvider.AuthenticateClient(ctx, parRequest.AuthenticateClientRequestBase, nil, &provider.AuthenticateClientOptions{
			AllowMissingDpopProof: true,
		})
		if err != nil {
			s.logger.Error("error authenticating client in standard request", "client_id", parRequest.ClientID, "error", err)
			return helpers.ServerError(e, to.StringPtr(err.Error()))
		}

		if parRequest.DpopJkt == nil {
			if client.Metadata.DpopBoundAccessTokens {
			}
		} else {
			if !client.Metadata.DpopBoundAccessTokens {
				msg := "dpop bound access tokens are not enabled for this client"
				return helpers.InputError(e, &msg)
			}
		}

		eat := time.Now().Add(constants.ParExpiresIn)
		id := oauth.GenerateRequestId()

		authRequest := &provider.OauthAuthorizationRequest{
			RequestId:  id,
			ClientId:   client.Metadata.ClientID,
			ClientAuth: *clientAuth,
			Parameters: parRequest,
			ExpiresAt:  eat,
		}

		if err := s.db.Create(ctx, authRequest, nil).Error; err != nil {
			s.logger.Error("error creating auth request in db", "error", err)
			return helpers.ServerError(e, nil)
		}

		reqUri = oauth.EncodeRequestUri(id)
		reqId = id
	}

	repo, _, err := s.getSessionRepoOrErr(e)
	if err != nil {
		return e.Redirect(303, "/account/signin?"+e.QueryParams().Encode())
	}

	var req provider.OauthAuthorizationRequest
	if err := s.db.Raw(ctx, "SELECT * FROM oauth_authorization_requests WHERE request_id = ?", nil, reqId).Scan(&req).Error; err != nil {
		return helpers.ServerError(e, to.StringPtr(err.Error()))
	}

	clientId := e.QueryParam("client_id")
	if clientId != req.ClientId {
		return helpers.InputError(e, to.StringPtr("client id does not match the client id for the supplied request"))
	}

	client, err := s.oauthProvider.ClientManager.GetClient(e.Request().Context(), req.ClientId)
	if err != nil {
		return helpers.ServerError(e, to.StringPtr(err.Error()))
	}

	scopes := strings.Split(req.Parameters.Scope, " ")
	appName := client.Metadata.ClientName

	data := map[string]any{
		"Scopes":      s.groupScopes(scopes),
		"AppName":     appName,
		"AppLogo":     client.Metadata.LogoURI,
		"RequestUri":  reqUri,
		"QueryParams": e.QueryParams().Encode(),
		"Handle":      repo.Actor.Handle,
	}

	return e.Render(200, "authorize.html", data)
}

type OauthAuthorizePostRequest struct {
	RequestUri    string `form:"request_uri"`
	AcceptOrRejct string `form:"accept_or_reject"`
}

func (s *Server) handleOauthAuthorizePost(e echo.Context) error {
	ctx := e.Request().Context()

	repo, _, err := s.getSessionRepoOrErr(e)
	if err != nil {
		return e.Redirect(303, "/account/signin")
	}

	var req OauthAuthorizePostRequest
	if err := e.Bind(&req); err != nil {
		s.logger.Error("error binding authorize post request", "error", err)
		return helpers.InputError(e, to.StringPtr("InvalidRequest"))
	}

	reqId, err := oauth.DecodeRequestUri(req.RequestUri)
	if err != nil {
		return helpers.InputError(e, to.StringPtr(err.Error()))
	}

	var authReq provider.OauthAuthorizationRequest
	if err := s.db.Raw(ctx, "SELECT * FROM oauth_authorization_requests WHERE request_id = ?", nil, reqId).Scan(&authReq).Error; err != nil {
		return helpers.ServerError(e, to.StringPtr(err.Error()))
	}

	if _, err := s.oauthProvider.ClientManager.GetClient(e.Request().Context(), authReq.ClientId); err != nil {
		return helpers.ServerError(e, to.StringPtr(err.Error()))
	}

	if req.AcceptOrRejct == "reject" {
		q := url.Values{}
		q.Set("error", "access_denied")
		q.Set("error_description", "The user denied the authorization request")
		q.Set("state", authReq.Parameters.State)
		return e.Redirect(303, authReq.Parameters.RedirectURI+"?"+q.Encode())
	}

	if time.Now().After(authReq.ExpiresAt) {
		return helpers.InputError(e, to.StringPtr("the request has expired"))
	}

	if authReq.Sub != nil || authReq.Code != nil {
		return helpers.InputError(e, to.StringPtr("this request was already authorized"))
	}

	code := oauth.GenerateCode()

	if err := s.db.Exec(ctx, "UPDATE oauth_authorization_requests SET sub = ?, code = ?, accepted = ?, ip = ? WHERE request_id = ?", nil, repo.Repo.Did, code, true, e.RealIP(), reqId).Error; err != nil {
		s.logger.Error("error updating authorization request", "error", err)
		return helpers.ServerError(e, nil)
	}

	q := url.Values{}
	q.Set("state", authReq.Parameters.State)
	q.Set("iss", "https://"+s.config.Hostname)
	q.Set("code", code)

	hashOrQuestion := "?"
	if authReq.Parameters.ResponseMode != nil {
		if *authReq.Parameters.ResponseMode == "fragment" {
			hashOrQuestion = "#"
		} else if *authReq.Parameters.ResponseMode == "query" {
		} else {
			if authReq.Parameters.ResponseType != "code" {
				hashOrQuestion = "#"
			}
		}
	} else {
		if authReq.Parameters.ResponseType != "code" {
			hashOrQuestion = "#"
		}
	}

	return e.Redirect(303, authReq.Parameters.RedirectURI+hashOrQuestion+q.Encode())
}
