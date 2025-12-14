package server

import (
	"fmt"
	"github.com/labstack/echo/v4"
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
	</style>
	<title>cocoon.scanash.com</title>
</head>
<body><pre>%s</pre></body>
</html>
`, strings.NewReplacer(
		"https://pdsmoover.com", `<a href="https://pdsmoover.com">https://pdsmoover.com</a>`,
		"https://ko-fi.com/scan", `<a href="https://ko-fi.com/scan">https://ko-fi.com/scan</a>`,
		"https://bsky.app/profile/scanash.com", `<a href="https://bsky.app/profile/scanash.com">https://bsky.app/profile/scanash.com</a>`,
		"https://github.com/scanash00/cocoon", `<a href="https://github.com/scanash00/cocoon">https://github.com/scanash00/cocoon</a>`,
	).Replace(plain)))
}
