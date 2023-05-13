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
	Description    string         `json:"description,omitempty"`
	Author         []string       `json:"author,omitempty"`
	Category       string         `json:"category,omitempty"`
	Publisher      string         `json:"publisher,omitempty"`
	Language       string         `json:"language,omitempty"`
	Series         string         `json:"series,omitempty"`
	Volume         int32          `json:"volume,omitempty"`
	Edition        string         `json:"edition,omitempty"`
	Year           int32          `json:"year,omitempty"`
	PageCount      int32          `json:"page_count,omitempty"`
	Isbn10         string         `json:"isbn_10,omitempty"`
	Isbn13         string         `json:"isbn_13,omitempty"`
	CoverPath      string         `json:"cover_path,omitempty"`
	S3FileKey      string         `json:"s3_file_key"`
	AdditionalInfo AdditionalInfo `json:"additional_info"`
	Version        int32          `json:"-"`
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
		edition = $9, year = $10, page_count = $11, isbn_10 = $12, isbn_13 = $13, cover_path = $14, version = version + 1
		WHERE id = $15 AND version = $16
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
		book.CoverPath,
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

func (m BookModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM book
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (m BookModel) GetAll(title string, author []string, isbn10, isbn13, publisher string, fromYear, toYear int, language, extension []string, filters Filters) ([]*Book, error) {
	query := `
		SELECT id, created_at, title, description, author, category, publisher,	language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, additional_info, version
		FROM book  
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '') 
		AND (author @> $2 OR $2 = '{}') 
		AND (LOWER(isbn_10) = LOWER($3) OR $3 = '') 
		AND (LOWER(isbn_13) = LOWER($4) OR $4 = '') 
		AND (to_tsvector('simple', publisher) @@ plainto_tsquery('simple', $5) OR $5 = '') 
		AND (
			CASE 
				WHEN $6 > 0 AND $7 = 0 THEN year BETWEEN $6 AND EXTRACT(YEAR FROM CURRENT_DATE)
				WHEN ($6 = 0 AND $7 > 0) OR ($6 > 0 AND $7 > 0) THEN year BETWEEN $6 AND $7
				ELSE year BETWEEN 1900 AND EXTRACT(YEAR FROM CURRENT_DATE)
			END
		)
		AND (language ILIKE ANY($8) OR $8 = '{}') 
		AND (additional_info::jsonb->>'FileExtension' ILIKE ANY($9) OR $9 = '{}')
		ORDER BY id`
	args := []interface{}{title, pq.Array(author), isbn10, isbn13, publisher, fromYear, toYear, pq.Array(language), pq.Array(extension)}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	books := []*Book{}
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&book.ID,
			&book.CreatedAt,
			&book.Title,
			&book.Description,
			pq.Array(book.Author),
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
			return nil, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return books, nil
}
