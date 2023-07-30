package data

import (
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

var ErrDuplicateBooklistFavourite = errors.New("duplicate booklist favourite")

// Booklist defines a booklist model.
type Booklist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Private     bool      `json:"private"`
	Content     Books     `json:"content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"-"`
}

type Books struct {
	Books    []*Book  `json:"books,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
}

func ValidateBooklist(v *validator.Validator, booklist *Booklist) {
	v.Check(booklist.Name != "", "name", "must be provided")
	v.Check(len(booklist.Name) <= 500, "name", "must not be more than 500 bytes long")
	v.Check(len(booklist.Description) <= 1000, "description", "must not be more than 1000 bytes long")
}
