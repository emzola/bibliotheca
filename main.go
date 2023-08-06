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
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jellydator/ttlcache/v3"
)

// app defines the application's layers and shared resources.
type app struct {
	config  config.Config
	repo    repository.Repository
	service service.Service
	handler *handler.Handler
}

// @title  Bibliotheca API
// @version 1.0.0
// @description This is an API service for book uploads and downloads.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email emma.idika@yahoo.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:4000
// @BasePath /
func main() {
	var cfg config.Config
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Initialize configuration
	err := cleanenv.ReadConfig("config.yml", &cfg)
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
