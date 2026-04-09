package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) HealthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok", "version": "0.1.0"})
}
