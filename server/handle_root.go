package server

import "github.com/labstack/echo/v4"

func (s *Server) handleRoot(e echo.Context) error {
	return e.Render(200, "index", map[string]interface{}{
		"Version": s.config.Version,
	})
}
