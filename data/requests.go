package data

import (
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// Request defines a book request.
type Request struct {
	ID        int64     `json:"id,omitempty"`
	UserID    int64     `json:"user_id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Isbn      string    `json:"isbn,omitempty"`
	Year      int32     `json:"year,omitempty"`
	Expiry    time.Time `json:"expiry,omitempty"`
	Status    string    `json:"status,omitempty"`
	Waitlist  int32     `json:"waitlist,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	Version   int32     `json:"-"`
}

func ValidateRequestIsbn(v *validator.Validator, isbn string) {
	v.Check(isbn != "", "isbn", "must be provided")
}
