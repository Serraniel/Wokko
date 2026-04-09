package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serraniel/wokko/backend/internal/config"
)

type Server struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

func NewServer(pool *pgxpool.Pool, cfg *config.Config) *Server {
	return &Server{pool: pool, cfg: cfg}
}
