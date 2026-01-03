package server

import (
	"context"
	"sync"
	"time"

	"github.com/haileyok/cocoon/oauth"
	"github.com/haileyok/cocoon/oauth/constants"
	"github.com/haileyok/cocoon/oauth/provider"
	"github.com/hako/durafmt"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleAccount(e echo.Context) error {
	ctx := e.Request().Context()
	logger := s.logger.With("name", "handleAuth")

	repo, sess, err := s.getSessionRepoOrErr(e)
	if err != nil {
		return e.Redirect(303, "/account/signin")
	}

	oldestPossibleSession := time.Now().Add(-1 * constants.ConfidentialClientSessionLifetime)

	var tokens []provider.OauthToken
	if err := s.db.Raw(ctx, "SELECT * FROM oauth_tokens WHERE sub = ? AND created_at > ? ORDER BY created_at ASC", nil, repo.Repo.Did, oldestPossibleSession).Scan(&tokens).Error; err != nil {
		logger.Error("couldnt fetch oauth sessions for account", "did", repo.Repo.Did, "error", err)
		sess.AddFlash("Unable to fetch sessions. See server logs for more details.", "error")
		sess.Save(e.Request(), e.Response())
		return e.Render(200, "account.html", map[string]any{
			"flashes": getFlashesFromSession(e, sess),
		})
	}

	var filtered []provider.OauthToken
	for _, t := range tokens {
		ageRes := oauth.GetSessionAgeFromToken(t)
		if ageRes.SessionExpired {
			continue
		}
		filtered = append(filtered, t)
	}

	clientNames := make(map[string]string)
	uniqueClients := make(map[string]struct{})
	for _, t := range filtered {
		uniqueClients[t.ClientId] = struct{}{}
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for cid := range uniqueClients {
		wg.Add(1)
		go func(clientId string) {
			defer wg.Done()
			tCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			name := clientId
			metadata, err := s.oauthProvider.ClientManager.GetClient(tCtx, clientId)
			if err == nil {
				name = metadata.Metadata.ClientName
			}
			mu.Lock()
			clientNames[clientId] = name
			mu.Unlock()
		}(cid)
	}
	wg.Wait()

	now := time.Now()

	tokenInfo := []map[string]string{}
	// Use filtered tokens only
	for _, t := range filtered {
		ageRes := oauth.GetSessionAgeFromToken(t)
		maxTime := constants.PublicClientSessionLifetime
		if t.ClientAuth.Method != "none" {
			maxTime = constants.ConfidentialClientSessionLifetime
		}

		clientName := t.ClientId
		if name, ok := clientNames[t.ClientId]; ok {
			clientName = name
		}

		tokenInfo = append(tokenInfo, map[string]string{
			"ClientName":  clientName,
			"Age":         durafmt.Parse(ageRes.SessionAge).LimitFirstN(2).String(),
			"LastUpdated": durafmt.Parse(now.Sub(t.UpdatedAt)).LimitFirstN(2).String(),
			"ExpiresIn":   durafmt.Parse(now.Add(maxTime).Sub(now)).LimitFirstN(2).String(),
			"Token":       t.Token,
			"Ip":          t.Ip,
		})
	}

	return e.Render(200, "account.html", map[string]any{
		"Repo":    repo,
		"Tokens":  tokenInfo,
		"flashes": getFlashesFromSession(e, sess),
	})
}
