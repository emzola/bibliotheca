package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/lib/pq"
)

// The Book struct contains the data fields for a book. The database expects NULL values
// for some fields when first creating a book - for such fields, we use a pointer type
// so that we can scan null values from the database into the Book struct.
type Book struct {
	ID int64 `json:"id"`
	// UserID         int64             `json:"-"`
	CreatedAt      time.Time      `json:"-"`
	Title          string         `json:"title"`
	Description    *string        `json:"description,omitempty"`
	Author         []string       `json:"author,omitempty"`
	Category       *string        `json:"category,omitempty"`
	Publisher      *string        `json:"publisher,omitempty"`
	Language       *string        `json:"language,omitempty"`
	Series         *string        `json:"series,omitempty"`
	Volume         *int32         `json:"volume,omitempty"`
	Edition        *string        `json:"edition,omitempty"`
	Year           *int32         `json:"year,omitempty"`
	PageCount      *int32         `json:"page_count,omitempty"`
	Isbn10         *string        `json:"isbn_10,omitempty"`
	Isbn13         *string        `json:"isbn_13,omitempty"`
	CoverPath      *string        `json:"cover_path,omitempty"`
	S3FileKey      string         `json:"s3_file_key"`
	AdditionalInfo AdditionalInfo `json:"additional_info"`
	Version        int32          `json:"-"`
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 500, "title", "must not be more than 500 bytes long")
	v.Check(*book.Description != "", "description", "must be provided")
	v.Check(len(*book.Description) <= 2000, "description", "must not be more than 2000 bytes long")
	v.Check(book.Author != nil, "author", "must be provided")
	v.Check(len(book.Author) >= 1, "author", "must contain at least 1 author")
	v.Check(len(book.Author) <= 5, "author", "must not contain more than 5 authors")
	v.Check(validator.Unique(book.Author), "author", "must not contain duplicate values")
	v.Check(*book.Category != "", "category", "must be provided")
	v.Check(*book.Language != "", "language", "must be provided")
	v.Check(*book.Year != 0, "year", "must be provided")
	v.Check(*book.Year >= 1900, "year", "must be greater than 1900")
	v.Check(*book.Year <= int32(time.Now().Year()), "year", "must not be in the future")
	v.Check(len(*book.Isbn10) <= 10, "isbn10", "must not be more than 10 characters")
	v.Check(len(*book.Isbn13) <= 13, "isbn13", "must not be more than 13 characters")
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

func (m BookModel) Get(id int64) (*Book, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, created_at, title, description, author, category, publisher,	language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, additional_info, version
		FROM book  
		WHERE id = $1`
	var book Book
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.CreatedAt,
		&book.Title,
		&book.Description,
		pq.Array(&book.Author),
		&book.Category,
		&book.Publisher,
		&book.Language,
		&book.Series,
		&book.Volume,
		&book.Edition,
		&book.Year,
		&book.PageCount,
		&book.Isbn10,
		&book.Isbn13,
		&book.CoverPath,
		&book.S3FileKey,
		&book.AdditionalInfo,
		&book.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &book, nil
}

func (m BookModel) Update(book *Book) error {
	query := `
		UPDATE book
		SET title = $1, description = $2, author = $3, category = $4, publisher = $5, language = $6, series = $7, volume = $8, 
		edition = $9, year = $10, page_count = $11, isbn_10 = $12, isbn_13 = $13, version = version + 1
		WHERE id = $14 AND version = $15
		RETURNING version`
	args := []interface{}{
		book.Title,
		book.Description,
		pq.Array(book.Author),
		book.Category,
		book.Publisher,
		book.Language,
		book.Series,
		book.Volume,
		book.Edition,
		book.Year,
		book.PageCount,
		book.Isbn10,
		book.Isbn13,
		book.ID,
		book.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&book.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}
