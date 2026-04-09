package api

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/auth"
)

func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
		}
		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := auth.ValidateAccessToken(s.cfg.JWTSecret, token)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		return next(c)
	}
}

func (s *Server) TandoorAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		var token string
		if strings.HasPrefix(header, "Bearer ") {
			token = strings.TrimPrefix(header, "Bearer ")
		} else if strings.HasPrefix(header, "Token ") {
			token = strings.TrimPrefix(header, "Token ")
		} else {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing token")
		}
		claims, err := auth.ValidateAccessToken(s.cfg.JWTSecret, token)
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		return next(c)
	}
}

func (s *Server) AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		role, _ := c.Get("role").(string)
		if role != "admin" {
			return echo.NewHTTPError(http.StatusForbidden, "admin role required")
		}
		return next(c)
	}
}
