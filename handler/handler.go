package handler

import (
	"github.com/emzola/bibliotheca/config"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	"github.com/emzola/bibliotheca/service"
	"github.com/jellydator/ttlcache/v3"
)

// Handler defines Handler layer.
type Handler struct {
	config  config.Config
	logger  *jsonlog.Logger
	cache   *ttlcache.Cache[string, int64]
	service service.Service
}

// New creates a new instance of Handler.
func New(cfg config.Config, logger *jsonlog.Logger, cache *ttlcache.Cache[string, int64], service service.Service) *Handler {
	return &Handler{
		config:  cfg,
		logger:  logger,
		cache:   cache,
		service: service,
	}
}
