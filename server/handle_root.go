package server

import (
	"fmt"
	"github.com/labstack/echo/v4"
)

func (s *Server) handleRoot(e echo.Context) error {
	return e.String(200, fmt.Sprintf(`

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

Feel free to join or migrate by using https://pdsmoover.com or goat-cli

Donate: https://ko-fi.com/scan
Follow Scan: https://bsky.app/profile/scanash.com
Code: https://github.com/scanash00/cocoon

Version: %s
`, s.config.Version))
}
