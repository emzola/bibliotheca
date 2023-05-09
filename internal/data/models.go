package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

// Models is a convenient single 'container' which holds and represents
// all database models for the application.
type Models struct {
	Book BookModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Book: BookModel{DB: db},
	}
}
