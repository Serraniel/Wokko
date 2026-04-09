package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/config"
)

func RegisterRoutes(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	s := NewServer(pool, cfg)
	e.GET("/api/health", s.HealthCheck)

	// Auth (public)
	authGroup := e.Group("/api/auth")
	authGroup.POST("/setup", s.SetupAdmin)
	authGroup.POST("/register", s.Register)
	authGroup.POST("/login", s.Login)
	authGroup.POST("/refresh", s.RefreshToken)
	authGroup.POST("/logout", s.Logout)
	authGroup.GET("/me", s.GetCurrentUser, s.AuthMiddleware)
	authGroup.GET("/passkey/register/begin", s.PasskeyRegisterBegin, s.AuthMiddleware)
	authGroup.POST("/passkey/register/finish", s.PasskeyRegisterFinish, s.AuthMiddleware)
	authGroup.GET("/passkey/login/begin", s.PasskeyLoginBegin)
	authGroup.POST("/passkey/login/finish", s.PasskeyLoginFinish)
}
