package main

import (
	"os"
	"sync"
	"time"

	"github.com/emzola/bibliotheca/config"
	"github.com/emzola/bibliotheca/handler"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	"github.com/emzola/bibliotheca/repository"
	"github.com/emzola/bibliotheca/repository/postgres.go"
	"github.com/emzola/bibliotheca/service"
	"github.com/jellydator/ttlcache/v3"
)

// app defines the application's layers and shared resources.
type app struct {
	config  config.Config
	repo    repository.Repository
	service service.Service
	handler *handler.Handler
}

func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Initialize configuration
	cfg, err := config.Decode()
	if err != nil {
		logger.PrintFatal(err, nil)
	}

	// Initialize database connection
	db, err := postgres.OpenDBConn(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	// Other shared resources: waitgroup and in-memory cache
	var wg sync.WaitGroup
	cache := ttlcache.New(ttlcache.WithTTL[string, int64](30 * time.Minute))
	go cache.Start()

	// Application layers
	repo := repository.New(db)
	service := service.New(cfg, &wg, logger, repo)
	handler := handler.New(cfg, logger, cache, service)

	// Instantiate application
	app := &app{
		config:  cfg,
		repo:    repo,
		service: service,
		handler: handler,
	}

	// Start HTTP server
	err = app.serve(&wg, logger)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}
