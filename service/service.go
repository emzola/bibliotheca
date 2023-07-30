package service

import (
	"sync"

	"github.com/emzola/bibliotheca/config"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	"github.com/emzola/bibliotheca/repository"
)

type Service interface {
	books
	reviews
	categories
	requests
	booklists
	comments
	users
	tokens
	failedValidation(map[string]string) error
}

// Services defines a service layer.
type service struct {
	config config.Config
	wg     sync.WaitGroup
	logger *jsonlog.Logger
	repo   repository.Repository
}

// New creates a new instance of Service.
func New(cfg config.Config, wg *sync.WaitGroup, logger *jsonlog.Logger, repo repository.Repository) *service {
	return &service{
		config: cfg,
		wg:     sync.WaitGroup{},
		logger: logger,
		repo:   repo,
	}
}
