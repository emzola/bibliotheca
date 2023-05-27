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
	Books     BookModel
	Reviews   ReviewModel
	Booklists BooklistModel
	// BooklistsBooks BooklistBookModel
	Tokens TokenModel
	Users  UserModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Books:     BookModel{DB: db},
		Reviews:   ReviewModel{DB: db},
		Booklists: BooklistModel{DB: db},
		// BooklistsBooks: BooklistBookModel{DB: db},
		Tokens: TokenModel{DB: db},
		Users:  UserModel{DB: db},
	}
}
