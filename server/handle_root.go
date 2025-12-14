package server

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"github.com/labstack/echo/v4"
	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/xrpc"
	"github.com/haileyok/cocoon/models"
	"strings"
)

func (s *Server) handleRoot(e echo.Context) error {
	plain := fmt.Sprintf(`

    _\/_
     /\
     /\
    /  \
    /~~\o
   /o   \
  /~~*~~~\
 o/    o \
 /~~~~~~~~\~
/__*_______\
     ||
   \====/
    \__/                    


This is an AT Protocol Personal Data Server (aka, an atproto PDS) hosted by Scan

Feel free to join or migrate by using https://pdsmoover.com or the Go Atproto CLI

Donate: https://ko-fi.com/scan
Follow Scan: https://bsky.app/profile/scanash.com
Code: https://github.com/scanash00/cocoon

Version: %s
`, s.config.Version)

	return e.HTML(200, fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<style>
		:root {
			color-scheme: light dark;
		}

		body {
			color: light-dark(#000000, #ffffff);
			background-color: light-dark(#ffffff, #18191b);
			font-family: monospace;
			margin: 25px;
		}

		pre {
			margin: 0;
			white-space: pre-wrap;
			word-wrap: break-word;
		}

		a {
			color: light-dark(#0a5dbd, #a4cefe);
		}

		.subtle {
			opacity: 0.8;
		}

		.posts {
			margin-top: 20px;
		}

		.post {
			margin: 14px 0;
			padding: 12px 14px;
			border: 1px solid light-dark(#e5e7eb, #2a2d31);
			border-radius: 10px;
			text-decoration: none;
			display: block;
		}

		.post:hover {
			border-color: light-dark(#c7ccd3, #3a3f45);
		}

		.post-meta {
			opacity: 0.8;
			font-size: 0.9em;
			margin-bottom: 6px;
			display: flex;
			align-items: center;
			gap: 10px;
		}

		.pfp {
			width: 22px;
			height: 22px;
			border-radius: 999px;
			background: light-dark(#e5e7eb, #2a2d31);
			flex: 0 0 22px;
			object-fit: cover;
		}

		.username a { text-decoration: none; }
		.username a:hover { text-decoration: underline; }

		.post-text {
			margin: 8px 0;
			white-space: pre-wrap;
			word-wrap: break-word;
		}

		.post-actions {
			margin-top: 10px;
			opacity: 0.7;
			font-size: 0.9em;
		}

		.counts {
			display: flex;
			gap: 16px;
			align-items: center;
		}

		.count {
			display: inline-flex;
			align-items: center;
			gap: 5px;
		}

		.icon {
			width: 16px;
			height: 16px;
			fill: currentColor;
			opacity: 0.7;
		}

		.replying {
			opacity: 0.7;
			font-size: 0.9em;
			margin-bottom: 6px;
		}

		.replying a {
			text-decoration: none;
		}

		.replying a:hover {
			text-decoration: underline;
		}
	</style>
	<title>cocoon.scanash.com</title>
</head>
<body>
	<pre>%s</pre>
	<div class="posts">
		<strong>Recent posts</strong>
		<div class="subtle">(from this PDS)</div>
		%s
	</div>
</body>
</html>
`, strings.NewReplacer(
		"https://pdsmoover.com", `<a href="https://pdsmoover.com">https://pdsmoover.com</a>`,
		"https://ko-fi.com/scan", `<a href="https://ko-fi.com/scan">https://ko-fi.com/scan</a>`,
		"https://bsky.app/profile/scanash.com", `<a href="https://bsky.app/profile/scanash.com">https://bsky.app/profile/scanash.com</a>`,
		"https://github.com/scanash00/cocoon", `<a href="https://github.com/scanash00/cocoon">https://github.com/scanash00/cocoon</a>`,
	).Replace(plain), s.renderRecentPostsHTML(e)))
}

func (s *Server) renderRecentPostsHTML(e echo.Context) string {
	const limit = 10

	type postRow struct {
		models.Record
		Handle string
	}

	var records []postRow
	err := s.db.Raw(
		"SELECT records.*, COALESCE(actors.handle, '') AS handle FROM records LEFT JOIN actors ON actors.did = records.did WHERE records.nsid = ? ORDER BY records.created_at DESC LIMIT ?",
		nil,
		"app.bsky.feed.post",
		limit,
	).Scan(&records).Error
	if err != nil {
		return `<div class="subtle">(failed to load posts)</div>`
	}

	if len(records) == 0 {
		return `<div class="subtle">(no posts yet)</div>`
	}

	postURIs := make([]string, 0, len(records))
	for _, r := range records {
		postURIs = append(postURIs, "at://"+r.Did+"/"+r.Nsid+"/"+r.Rkey)
	}

	postsByURI, err := s.fetchAppViewPosts(e.Request().Context(), postURIs)
	if err != nil {
		postsByURI = nil
	}

	var b strings.Builder
	for _, r := range records {
		val, err := atdata.UnmarshalCBOR(r.Value)
		if err != nil {
			continue
		}

		text, _ := val["text"].(string)
		postURI := "at://" + r.Did + "/" + r.Nsid + "/" + r.Rkey

		authorHandle := r.Handle
		authorAvatar := ""
		replyCount := int64(0)
		repostCount := int64(0)
		likeCount := int64(0)
		postLink := ""
		replyingTo := ""

		if postsByURI != nil {
			if v, ok := postsByURI[postURI]; ok && v.Post != nil {
				authorHandle = v.Post.Author.Handle
				authorAvatar = toStringAny(any(v.Post.Author.Avatar))
				replyCount = toInt64Any(any(v.Post.ReplyCount))
				repostCount = toInt64Any(any(v.Post.RepostCount))
				likeCount = toInt64Any(any(v.Post.LikeCount))

				if v.Post.Uri != "" {
					postLink = toBskyPostLink(v.Post.Author.Handle, v.Post.Uri)
				}
				if v.ReplyingToHandle != "" {
					replyingTo = v.ReplyingToHandle
				}
			}
		}

		if postLink == "" {
			postLink = toBskyPostLink(authorHandle, postURI)
		}

		b.WriteString(`<div class="post" onclick="window.location.href='` + html.EscapeString(postLink) + `'" style="cursor:pointer">`)
		if replyingTo != "" {
			b.WriteString(`<div class="replying">Replying to <a href="` + html.EscapeString(toBskyProfileLink(replyingTo)) + `" onclick="event.stopPropagation()">@` + html.EscapeString(replyingTo) + `</a></div>`)
		}
		b.WriteString(`<div class="post-meta">`)
		if authorAvatar != "" {
			b.WriteString(`<img class="pfp" src="` + html.EscapeString(authorAvatar) + `" alt="" />`)
		} else {
			b.WriteString(`<div class="pfp"></div>`)
		}
		b.WriteString(`<span class="username">`)
		if authorHandle != "" {
			b.WriteString(`<a href="` + html.EscapeString(toBskyProfileLink(authorHandle)) + `" onclick="event.stopPropagation()">@` + html.EscapeString(authorHandle) + `</a>`)
		} else {
			b.WriteString(html.EscapeString(r.Did))
		}
		b.WriteString(`</span>`)
		b.WriteString(`</div>`)
		b.WriteString(`<div class="post-text">`)
		if text == "" {
			b.WriteString(`<span class="subtle">(no text)</span>`)
		} else {
			b.WriteString(html.EscapeString(text))
		}
		b.WriteString(`</div>`)
		b.WriteString(`<div class="post-actions">`)
		b.WriteString(`<div class="counts">`)
		b.WriteString(fmt.Sprintf(`<span class="count"><svg class="icon" viewBox="0 0 24 24"><path d="M21 6h-2v9H6v2c0 .55.45 1 1 1h11l4 4V7c0-.55-.45-1-1-1zm-4 6V3c0-.55-.45-1-1-1H3c-.55 0-1 .45-1 1v14l4-4h10c.55 0 1-.45 1-1z"/></svg>%d</span>`, replyCount))
		b.WriteString(fmt.Sprintf(`<span class="count"><svg class="icon" viewBox="0 0 24 24"><path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z"/></svg>%d</span>`, repostCount))
		b.WriteString(fmt.Sprintf(`<span class="count"><svg class="icon" viewBox="0 0 24 24"><path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/></svg>%d</span>`, likeCount))
		b.WriteString(`</div>`)
		b.WriteString(`</div>`)
		b.WriteString(`</div>`)
	}

	if b.Len() == 0 {
		return `<div class="subtle">(no readable posts)</div>`
	}

	return b.String()
}


type appViewPost struct {
	Post             *bsky.FeedDefs_PostView
	ReplyingToHandle string
}

func (s *Server) fetchAppViewPosts(ctx context.Context, postURIs []string) (map[string]appViewPost, error) {
	if s.config.FallbackProxy == "" {
		return nil, fmt.Errorf("no fallback proxy configured")
	}

	endpoint, err := s.getFallbackProxyEndpoint(ctx)
	if err != nil {
		return nil, err
	}

	cli := xrpc.Client{Host: endpoint}
	resp, err := bsky.FeedGetPosts(ctx, &cli, postURIs)
	if err != nil {
		return nil, err
	}

	posts := make(map[string]appViewPost, len(resp.Posts))
	for _, p := range resp.Posts {
		posts[p.Uri] = appViewPost{Post: p}
	}

	parentURIs := make([]string, 0)
	for _, p := range resp.Posts {
		if p.Record == nil {
			continue
		}
		post, ok := p.Record.Val.(*bsky.FeedPost)
		if !ok || post == nil || post.Reply == nil || post.Reply.Parent == nil {
			continue
		}
		if post.Reply.Parent.Uri != "" {
			parentURIs = append(parentURIs, post.Reply.Parent.Uri)
		}
	}

	if len(parentURIs) > 0 {
		parentsResp, err := bsky.FeedGetPosts(ctx, &cli, parentURIs)
		if err == nil {
			parentByURI := make(map[string]*bsky.FeedDefs_PostView, len(parentsResp.Posts))
			for _, p := range parentsResp.Posts {
				parentByURI[p.Uri] = p
			}
			for uri, entry := range posts {
				if entry.Post == nil || entry.Post.Record == nil {
					continue
				}
				post, ok := entry.Post.Record.Val.(*bsky.FeedPost)
				if !ok || post == nil || post.Reply == nil || post.Reply.Parent == nil {
					continue
				}
				puri := post.Reply.Parent.Uri
				if puri == "" {
					continue
				}
				if pv := parentByURI[puri]; pv != nil {
					entry.ReplyingToHandle = pv.Author.Handle
					posts[uri] = entry
				}
			}
		}
	}

	return posts, nil
}

func (s *Server) getFallbackProxyEndpoint(ctx context.Context) (string, error) {
	pts := strings.Split(s.config.FallbackProxy, "#")
	if len(pts) != 2 {
		return "", fmt.Errorf("invalid fallback proxy")
	}
	svcDid := pts[0]
	svcId := "#" + pts[1]

	doc, err := s.passport.FetchDoc(ctx, svcDid)
	if err != nil {
		return "", err
	}

	for _, svc := range doc.Service {
		if svc.Id == svcId {
			return strings.TrimPrefix(svc.ServiceEndpoint, "https://"), nil
		}
	}

	return "", fmt.Errorf("fallback proxy service not found")
}

func toInt64Any(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return int64(t)
	case int64:
		return t
	case *int64:
		if t == nil {
			return 0
		}
		return *t
	case uint64:
		return int64(t)
	case *uint64:
		if t == nil {
			return 0
		}
		return int64(*t)
	default:
		return 0
	}
}

func toStringAny(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case *string:
		if t == nil {
			return ""
		}
		return *t
	default:
		return ""
	}
}

func toBskyProfileLink(handle string) string {
	return "https://bsky.app/profile/" + url.PathEscape(handle)
}

func toBskyPostLink(handle string, atURI string) string {
	parts := strings.Split(atURI, "/")
	if len(parts) < 5 {
		return "https://bsky.app/profile/" + url.PathEscape(handle)
	}
	rkey := parts[len(parts)-1]
	return "https://bsky.app/profile/" + url.PathEscape(handle) + "/post/" + url.PathEscape(rkey)
}
