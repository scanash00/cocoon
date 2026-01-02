package server

import (
	"strings"
)

type scopeInfo struct {
	Name        string
	Description string
	Icon        string
}

var scopeDescriptions = map[string]scopeInfo{
	"atproto": {
		Name:        "ATProto Access",
		Description: "Confirm you're using an ATProto account",
		Icon:        "key",
	},
	"transition:generic": {
		Name:        "Full App Access",
		Description: "Read and write posts, upload media, access preferences",
		Icon:        "shield",
	},
	"transition:chat.bsky": {
		Name:        "Direct Messages",
		Description: "Read and send direct messages",
		Icon:        "message",
	},
	"transition:email": {
		Name:        "Email Address",
		Description: "View your email address",
		Icon:        "mail",
	},
}

func (s *Server) parseScopeForDisplay(scope string) scopeInfo {
	if info, ok := scopeDescriptions[scope]; ok {
		return info
	}

	if strings.HasPrefix(scope, "repo:") {
		collection := strings.TrimPrefix(scope, "repo:")
		if idx := strings.Index(collection, "?"); idx != -1 {
			collection = collection[:idx]
		}
		if collection == "*" {
			return scopeInfo{
				Name:        "Repository Access",
				Description: "Read and write all record types",
				Icon:        "database",
			}
		}
		parts := strings.Split(collection, ".")
		if len(parts) > 0 {
			shortName := parts[len(parts)-1]
			return scopeInfo{
				Name:        "Write " + shortName + " records",
				Description: "Create, update, or delete " + collection,
				Icon:        "edit",
			}
		}
		return scopeInfo{
			Name:        "Repository Access",
			Description: "Access to " + collection,
			Icon:        "database",
		}
	}

	if strings.HasPrefix(scope, "rpc:") {
		method := strings.TrimPrefix(scope, "rpc:")
		if idx := strings.Index(method, "?"); idx != -1 {
			method = method[:idx]
		}
		parts := strings.Split(method, ".")
		if len(parts) > 0 {
			shortName := parts[len(parts)-1]
			return scopeInfo{
				Name:        "API: " + shortName,
				Description: "Call remote API endpoints",
				Icon:        "globe",
			}
		}
		return scopeInfo{
			Name:        "API Access",
			Description: "Make authenticated API requests",
			Icon:        "globe",
		}
	}

	if strings.HasPrefix(scope, "blob:") {
		mimeType := strings.TrimPrefix(scope, "blob:")
		if mimeType == "*/*" {
			return scopeInfo{
				Name:        "Upload Files",
				Description: "Upload any file type",
				Icon:        "upload",
			}
		}
		if strings.HasPrefix(mimeType, "image/") {
			return scopeInfo{
				Name:        "Upload Images",
				Description: "Upload image files",
				Icon:        "image",
			}
		}
		if strings.HasPrefix(mimeType, "video/") {
			return scopeInfo{
				Name:        "Upload Videos",
				Description: "Upload video files",
				Icon:        "video",
			}
		}
		if strings.HasPrefix(mimeType, "audio/") {
			return scopeInfo{
				Name:        "Upload Audio",
				Description: "Upload audio files",
				Icon:        "music",
			}
		}
		if strings.HasPrefix(mimeType, "text/") {
			return scopeInfo{
				Name:        "Upload Text",
				Description: "Upload text files",
				Icon:        "file-text",
			}
		}
		return scopeInfo{
			Name:        "Upload Files",
			Description: "Upload specific file types",
			Icon:        "upload",
		}
	}

	if strings.HasPrefix(scope, "account:") {
		attr := strings.TrimPrefix(scope, "account:")
		if idx := strings.Index(attr, "?"); idx != -1 {
			hasManage := strings.Contains(scope, "action=manage")
			attr = attr[:idx]
			if attr == "email" {
				if hasManage {
					return scopeInfo{
						Name:        "Manage Email",
						Description: "View and change your email address",
						Icon:        "mail",
					}
				}
				return scopeInfo{
					Name:        "View Email",
					Description: "See your email address",
					Icon:        "mail",
				}
			}
			if attr == "repo" && hasManage {
				return scopeInfo{
					Name:        "Import Repository",
					Description: "Import account data (for migration)",
					Icon:        "download",
				}
			}
		}
		if attr == "email" {
			return scopeInfo{
				Name:        "View Email",
				Description: "See your email address",
				Icon:        "mail",
			}
		}
		if attr == "repo" {
			return scopeInfo{
				Name:        "Repository Management",
				Description: "Manage your data repository",
				Icon:        "database",
			}
		}
		if strings.HasPrefix(attr, "status") {
			if strings.Contains(scope, "action=manage") {
				return scopeInfo{
					Name:        "Account Status",
					Description: "Activate or deactivate your account",
					Icon:        "power",
				}
			}
			return scopeInfo{
				Name:        "View Account Status",
				Description: "Check account activation status",
				Icon:        "info",
			}
		}
		return scopeInfo{
			Name:        "Account: " + attr,
			Description: "Access account settings",
			Icon:        "settings",
		}
	}

	if strings.HasPrefix(scope, "identity:") {
		attr := strings.TrimPrefix(scope, "identity:")
		if attr == "*" {
			return scopeInfo{
				Name:        "Full Identity Control",
				Description: "Change handle and DID document",
				Icon:        "user-cog",
			}
		}
		if attr == "handle" {
			return scopeInfo{
				Name:        "Change Handle",
				Description: "Update your username",
				Icon:        "at-sign",
			}
		}
		return scopeInfo{
			Name:        "Identity: " + attr,
			Description: "Modify your identity",
			Icon:        "user",
		}
	}

	if strings.HasPrefix(scope, "include:") {
		setName := strings.TrimPrefix(scope, "include:")
		parts := strings.Split(setName, ".")
		if len(parts) > 0 {
			shortName := parts[len(parts)-1]
			return scopeInfo{
				Name:        shortName,
				Description: "App-specific permissions",
				Icon:        "package",
			}
		}
		return scopeInfo{
			Name:        "Extended Permissions",
			Description: "Additional app permissions",
			Icon:        "package",
		}
	}

	return scopeInfo{
		Name:        scope,
		Description: "",
		Icon:        "circle",
	}
}

func (s *Server) groupScopes(scopes []string) []scopeInfo {
	seen := make(map[string]bool)
	var result []scopeInfo

	for _, scope := range scopes {
		info := s.parseScopeForDisplay(scope)

		key := info.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, info)
	}

	return result
}
