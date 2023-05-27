package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/lib/pq"
)

var ErrDuplicateFavourite = errors.New("duplicate favourite")

// The Book struct contains the data fields for a book.
type Book struct {
	ID          int64     `json:"id"`
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

// The BookModel struct wraps a sql.DB connection pool for Book.
type BookModel struct {
	DB *sql.DB
}

func (m BookModel) Insert(book *Book) error {
	query := `
		INSERT INTO books (user_id, title, s3_file_key, fname, extension, size)	
		VALUES ($1, $2, $3, $4, $5, $6)
	  	RETURNING id, created_at, version`
	args := []interface{}{book.UserID, book.Title, book.S3FileKey, book.Filename, book.Extension, book.Size}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}

func (m BookModel) Get(id int64) (*Book, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, user_id, created_at, title, description, author, category, publisher, language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, fname, extension, size, popularity, version
		FROM books 
		WHERE id = $1`
	var book Book
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
		&book.UserID,
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
		&book.Filename,
		&book.Extension,
		&book.Size,
		&book.Popularity,
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
		UPDATE books
		SET title = $1, description = $2, author = $3, category = $4, publisher = $5, language = $6, series = $7, volume = $8, 
		edition = $9, year = $10, page_count = $11, isbn_10 = $12, isbn_13 = $13, cover_path = $14, popularity = $15, version = version + 1
		WHERE id = $16 AND version = $17
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
		book.Popularity,
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
		DELETE FROM books
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

func (m BookModel) GetAll(title string, author []string, isbn10, isbn13, publisher string, fromYear, toYear int, language, extension []string, filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, created_at, title, description, author, category, publisher, language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, fname, extension, size, popularity, version
		FROM books  
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
		AND (extension ILIKE ANY($9) OR $9 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $10 OFFSET $11`,
		filters.sortColumn(), filters.sortDirection(),
	)
	args := []interface{}{
		title,
		pq.Array(author),
		isbn10,
		isbn13,
		publisher,
		fromYear,
		toYear,
		pq.Array(language),
		pq.Array(extension),
		filters.limit(),
		filters.offset(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	books := []*Book{}
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.UserID,
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
			&book.Filename,
			&book.Extension,
			&book.Size,
			&book.Popularity,
			&book.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

func (m BookModel) AddFavouriteForUser(userID, bookID int64) error {
	query := `
		INSERT INTO users_favouritebooks (user_id, book_id)
		VALUES ($1, $2)`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_favouritebooks_pkey"`:
			return ErrDuplicateFavourite
		default:
			return err
		}
	}
	return nil
}

func (m BookModel) RemoveFavouriteForUser(userID, bookID int64) error {
	if userID < 1 || bookID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_favouritebooks
		WHERE user_id = $1 AND book_id = $2`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := m.DB.ExecContext(ctx, query, args...)
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

func (m BookModel) GetAllFavouritesForUser(userID int64, filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
		FROM books
		INNER JOIN users_favouritebooks ON users_favouritebooks.book_id = books.id
		INNER JOIN users ON users_favouritebooks.user_id = users.id
		WHERE users.id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.sortColumn(), filters.sortDirection(),
	)
	args := []interface{}{userID, filters.limit(), filters.offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()
	defer rows.Close()
	totalRecords := 0
	books := []*Book{}
	for rows.Next() {
		var book Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.UserID,
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
			&book.Filename,
			&book.Extension,
			&book.Size,
			&book.Popularity,
			&book.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}
