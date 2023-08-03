package data

import (
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

const (
	ScopeCover = "cover"
	ScopeBook  = "book"
)

const DailyDownloadLimit int8 = 10

// Book defines a book model.
type Book struct {
	ID          int64     `json:"id" `
	UserID      int64     `json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Author      []string  `json:"author,omitempty"`
	Category    string    `json:"category,omitempty"`
	Publisher   string    `json:"publisher,omitempty"`
	Language    string    `json:"language,omitempty"`
	Series      string    `json:"series,omitempty"`
	Volume      int32     `json:"volume,omitempty"`
	Edition     string    `json:"edition,omitempty"`
	Year        int32     `json:"year,omitempty"`
	PageCount   int32     `json:"page_count,omitempty"`
	Isbn10      string    `json:"isbn_10,omitempty"`
	Isbn13      string    `json:"isbn_13,omitempty"`
	CoverPath   string    `json:"cover_path,omitempty"`
	S3FileKey   string    `json:"s3_file_key"`
	Filename    string    `json:"filename"`
	Extension   string    `json:"extension"`
	Size        int64     `json:"size"`
	Popularity  float64   `json:"popularity,omitempty"`
	Version     int32     `json:"-"`
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(book.Description != "", "description", "must be provided")
	v.Check(len(book.Description) <= 2000, "description", "must not be more than 2000 bytes long")
	v.Check(book.Author != nil, "author", "must be provided")
	v.Check(len(book.Author) >= 1, "author", "must contain at least 1 author")
	v.Check(len(book.Author) <= 5, "author", "must not contain more than 5 authors")
	v.Check(validator.Unique(book.Author), "author", "must not contain duplicate values")
	v.Check(book.Category != "", "category", "must be provided")
	v.Check(book.Language != "", "language", "must be provided")
	v.Check(book.Year != 0, "year", "must be provided")
	v.Check(book.Year >= 1900, "year", "must be greater than 1900")
	v.Check(book.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(len(book.Isbn10) <= 13, "isbn10", "must not be more than 13 characters")
	v.Check(len(book.Isbn13) <= 17, "isbn13", "must not be more than 17 characters")
}
