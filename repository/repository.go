package repository

import (
	"database/sql"
)

type Repository interface {
	books
	reviews
	categories
	requests
	booklists
	comments
	users
	tokens
}

// Repository defines the app's repository layer.
type repository struct {
	db *sql.DB
}

// New creates a new instance of Repository.
func New(db *sql.DB) *repository {
	return &repository{db: db}
}
