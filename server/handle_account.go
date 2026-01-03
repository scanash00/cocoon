package server

import (
	"context"
	"sort"
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

	type Session struct {
		ClientName  string
		Age         string
		LastUpdated string
		ExpiresIn   string
		Token       string
		Ip          string
	}

	type ClientGroup struct {
		ID       string
		Name     string
		Sessions []Session
	}
	
	now := time.Now()

	groupsMap := make(map[string][]Session)

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

		session := Session{
			ClientName:  clientName,
			Age:         durafmt.Parse(ageRes.SessionAge).LimitFirstN(2).String(),
			LastUpdated: durafmt.Parse(now.Sub(t.UpdatedAt)).LimitFirstN(2).String(),
			ExpiresIn:   durafmt.Parse(now.Add(maxTime).Sub(now)).LimitFirstN(2).String(),
			Token:       t.Token,
			Ip:          t.Ip,
		}

		groupsMap[t.ClientId] = append(groupsMap[t.ClientId], session)
	}

	var clientGroups []ClientGroup
	for clientId, sessions := range groupsMap {
		name := clientId
		if n, ok := clientNames[clientId]; ok {
			name = n
		}
		
		clientGroups = append(clientGroups, ClientGroup{
			ID:       clientId,
			Name:     name,
			Sessions: sessions,
		})
	}

	sort.Slice(clientGroups, func(i, j int) bool {
		return clientGroups[i].Name < clientGroups[j].Name
	})

	return e.Render(200, "account.html", map[string]any{
		"Repo":         repo,
		"ClientGroups": clientGroups,
		"flashes":      getFlashesFromSession(e, sess),
	})
}
