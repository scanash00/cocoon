package server

import (
	"fmt"
	"html"
	"github.com/labstack/echo/v4"
	"github.com/bluesky-social/indigo/atproto/atdata"
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
		}

		.post-meta {
			opacity: 0.8;
			font-size: 0.9em;
			margin-bottom: 6px;
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
	likeCounts := s.countLikeCountsForPostURIs(e, postURIs)

	var b strings.Builder
	for _, r := range records {
		val, err := atdata.UnmarshalCBOR(r.Value)
		if err != nil {
			continue
		}

		text, _ := val["text"].(string)
		createdAt, _ := val["createdAt"].(string)
		postURI := "at://" + r.Did + "/" + r.Nsid + "/" + r.Rkey
		likes := likeCounts[postURI]

		b.WriteString(`<div class="post">`)
		b.WriteString(`<div class="post-meta">`)
		if r.Handle != "" {
			b.WriteString(html.EscapeString(r.Handle))
		} else {
			b.WriteString(html.EscapeString(r.Did))
		}
		if createdAt != "" {
			b.WriteString(` &middot; `)
			b.WriteString(html.EscapeString(createdAt))
		}
		b.WriteString(` &middot; `)
		b.WriteString(fmt.Sprintf("%d likes", likes))
		b.WriteString(`</div>`)
		b.WriteString(`<div>`)
		if text == "" {
			b.WriteString(`<span class="subtle">(no text)</span>`)
		} else {
			b.WriteString(html.EscapeString(text))
		}
		b.WriteString(`</div>`)
		b.WriteString(`</div>`)
	}

	if b.Len() == 0 {
		return `<div class="subtle">(no readable posts)</div>`
	}

	return b.String()
}

func (s *Server) countLikeCountsForPostURIs(e echo.Context, postURIs []string) map[string]int {
	counts := make(map[string]int, len(postURIs))
	if len(postURIs) == 0 {
		return counts
	}

	set := make(map[string]struct{}, len(postURIs))
	for _, uri := range postURIs {
		set[uri] = struct{}{}
	}

	var likes []models.Record
	if err := s.db.Raw(
		"SELECT * FROM records WHERE nsid = ? ORDER BY created_at DESC LIMIT ?",
		nil,
		"app.bsky.feed.like",
		5000,
	).Scan(&likes).Error; err != nil {
		return counts
	}

	for _, r := range likes {
		val, err := atdata.UnmarshalCBOR(r.Value)
		if err != nil {
			continue
		}
		subject, ok := val["subject"].(map[string]any)
		if !ok {
			continue
		}
		subURI, _ := subject["uri"].(string)
		if subURI == "" {
			continue
		}
		if _, ok := set[subURI]; !ok {
			continue
		}
		counts[subURI]++
	}

	return counts
}
