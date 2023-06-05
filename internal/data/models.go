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
	Books      BookModel
	Categories CategoryModel
	Reviews    ReviewModel
	Booklists  BooklistModel
	Comments   CommentModel
	Requests   RequestModel
	Tokens     TokenModel
	Users      UserModel
}

func NewModels(db *sql.DB) *Models {
	return &Models{
		Books:      BookModel{DB: db},
		Categories: CategoryModel{DB: db},
		Reviews:    ReviewModel{DB: db},
		Booklists:  BooklistModel{DB: db},
		Comments:   CommentModel{DB: db},
		Requests:   RequestModel{DB: db},
		Tokens:     TokenModel{DB: db},
		Users:      UserModel{DB: db},
	}
}
