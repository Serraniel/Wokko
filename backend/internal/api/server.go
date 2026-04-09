package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/serraniel/wokko/backend/internal/config"
	"github.com/serraniel/wokko/backend/internal/storage"
)

type Server struct {
	pool    *pgxpool.Pool
	cfg     *config.Config
	storage *storage.Storage
}

func NewServer(pool *pgxpool.Pool, cfg *config.Config) *Server {
	return &Server{
		pool:    pool,
		cfg:     cfg,
		storage: storage.New(cfg.UploadPath),
	}
}
