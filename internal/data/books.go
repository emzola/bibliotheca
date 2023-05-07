package data

import (
	"context"
	"database/sql"
	"time"
)

// Book contains the data fields and types for a book.
type Book struct {
	ID int64 `json:"id"`
	// UserID         int64             `json:"-"`
	CreatedAt      time.Time      `json:"-"`
	Title          string         `json:"title"`
	Description    string         `json:"description,omitempty"`
	Author         []string       `json:"author,omitempty"`
	Category       string         `json:"category,omitempty"`
	Publisher      string         `json:"publisher,omitempty"`
	Language       string         `json:"language,omitempty"`
	Series         string         `json:"series,omitempty"`
	Volume         int32          `json:"volume,omitempty"`
	Edition        int32          `json:"edition,omitempty"`
	Year           int32          `json:"year,omitempty"`
	PageCount      int32          `json:"page_count,omitempty"`
	Isbn10         string         `json:"isbn_10,omitempty"`
	Isbn13         string         `json:"isbn_13,omitempty"`
	CoverUrl       string         `json:"cover_url,omitempty"`
	S3FileKey      string         `json:"s3_file_key"`
	AdditionalInfo AdditionalInfo `json:"additional_info"` // original filename and size (in KB or MB)
	Version        int32          `json:"-"`
}

// BookModel wraps a sql.DB connection pool for Book.
type BookModel struct {
	DB *sql.DB
}

func (m BookModel) Insert(book *Book) error {
	query := `
		INSERT INTO book (title,  s3_file_key, additional_info)	VALUES ($1, $2, $3)
	  	RETURNING id, created_at, version`
	args := []interface{}{book.Title, book.S3FileKey, book.AdditionalInfo}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}
