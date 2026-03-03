package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PermGroupDisplay represents a grouped permission within a permission-set scope.
type PermGroupDisplay struct {
	ActionsStr  string
	Resource    string
	Collections []string
}

type lexiconSchemaResponse struct {
	ID   string                      `json:"id"`
	Defs map[string]lexiconSchemaDef `json:"defs"`
}

type lexiconSchemaDef struct {
	Type        string              `json:"type"`
	Title       string              `json:"title"`
	Detail      string              `json:"detail"`
	Permissions []lexiconPermission `json:"permissions"`
}

type lexiconPermission struct {
	Action     []string `json:"action"`
	Resource   string   `json:"resource"`
	Collection []string `json:"collection"`
}

func isPermSetScope(scope string) bool {
	return strings.Contains(scope, ".") && !strings.Contains(scope, ":")
}

func nsidToAuthorityDomain(nsid string) string {
	parts := strings.Split(nsid, ".")
	if len(parts) < 2 {
		return ""
	}
	authority := make([]string, len(parts)-1)
	copy(authority, parts[:len(parts)-1])
	for i, j := 0, len(authority)-1; i < j; i, j = i+1, j-1 {
		authority[i], authority[j] = authority[j], authority[i]
	}
	return strings.Join(authority, ".")
}

func fetchPermSetLexicon(ctx context.Context, cli *http.Client, nsid string) (*lexiconSchemaDef, error) {
	domain := nsidToAuthorityDomain(nsid)
	if domain == "" {
		return nil, fmt.Errorf("could not derive authority domain from nsid %q", nsid)
	}

	fetchURL := fmt.Sprintf("https://%s/xrpc/com.atproto.lexicon.schema?id=%s", domain, nsid)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lexicon fetch returned status %d", resp.StatusCode)
	}

	var schema lexiconSchemaResponse
	if err := json.NewDecoder(resp.Body).Decode(&schema); err != nil {
		return nil, err
	}

	main, ok := schema.Defs["main"]
	if !ok {
		return nil, fmt.Errorf("lexicon schema has no 'main' def")
	}
	if main.Type != "permission-set" {
		return nil, fmt.Errorf("lexicon def type is %q, expected permission-set", main.Type)
	}

	return &main, nil
}

func splitScopes(ctx context.Context, cli *http.Client, scopes []string) []scopeInfo {
	seen := make(map[string]bool)
	var result []scopeInfo

	for _, scope := range scopes {
		info := parseScopeForDisplayWithNSID(scope)

		key := info.Name
		if seen[key] {
			continue
		}
		seen[key] = true

		if info.IsPermSet {
			def, err := fetchPermSetLexicon(ctx, cli, scope)
			if err != nil {
				info.FetchFailed = true
			} else {
				info.Name = def.Title
				info.PermNSID = scope
				info.PermDetail = def.Detail
				for _, perm := range def.Permissions {
					info.PermGroups = append(info.PermGroups, PermGroupDisplay{
						ActionsStr:  strings.Join(perm.Action, ", "),
						Resource:    perm.Resource,
						Collections: perm.Collection,
					})
				}
			}
		}

		result = append(result, info)
	}
	return result
}
