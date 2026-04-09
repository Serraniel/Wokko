package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/serraniel/wokko/backend/internal/config"
)

func RegisterRoutes(e *echo.Echo, pool *pgxpool.Pool, cfg *config.Config) {
	s := NewServer(pool, cfg)
	e.GET("/api/health", s.HealthCheck)
	// Routes will be added in subsequent tasks
}
