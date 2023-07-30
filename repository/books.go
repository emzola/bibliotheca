package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/bibliotheca/data"
	"github.com/lib/pq"
)

type books interface {
	CreateBook(book *data.Book) error
	GetBook(ID int64) (*data.Book, error)
	GetAllBooks(search string, fromYear, toYear int, language, extension []string, filters data.Filters) ([]*data.Book, data.Metadata, error)
	UpdateBook(book *data.Book) error
	DeleteBook(bookID int64) error
	AddDownloadForUser(userID int64, bookID int64) error
	RemoveDownloadForUser(userID int64, bookID int64) error
	FavouriteBook(userID int64, bookID int64) error
	DeleteFavouriteBook(userID int64, bookID int64) error
}

// CreateBook creates a new book record.
func (r *repository) CreateBook(book *data.Book) error {
	query := `
			INSERT INTO books (user_id, title, s3_file_key, fname, extension, size)
			VALUES ($1, $2, $3, $4, $5, $6)
		  	RETURNING id, created_at, version`
	args := []interface{}{book.UserID, book.Title, book.S3FileKey, book.Filename, book.Extension, book.Size}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&book.ID, &book.CreatedAt, &book.Version)
}

// GetBook retrieves a book record by its ID.
func (r *repository) GetBook(ID int64) (*data.Book, error) {
	if ID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, user_id, created_at, title, description, author, category, publisher, language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, fname, extension, size, popularity, version
		FROM books 
		WHERE id = $1`
	var book data.Book
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, ID).Scan(
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

// GetAllBooks retrieves retrieves a paginated list of all book records.
// Records can be filtered and sorted.
func (r *repository) GetAllBooks(search string, fromYear, toYear int, language, extension []string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, created_at, title, description, author, category, publisher, language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, fname, extension, size, popularity, version
		FROM books  
		WHERE (
			to_tsvector('simple', title) || 
			to_tsvector(array_to_string(author,' '::text)) ||
			to_tsvector('simple', isbn_10) || 
			to_tsvector('simple', isbn_13) || 
			to_tsvector('simple', publisher) 
			@@ plainto_tsquery('simple', $1) OR $1 = ''
		) 
		AND (
			CASE 
				WHEN $2 > 0 AND $3 = 0 THEN year BETWEEN $2 AND EXTRACT(YEAR FROM CURRENT_DATE)
				WHEN ($2 = 0 AND $3 > 0) OR ($2 > 0 AND $3 > 0) THEN year BETWEEN $2 AND $3
				ELSE year BETWEEN 1900 AND EXTRACT(YEAR FROM CURRENT_DATE)
			END
		)
		AND (language ILIKE ANY($4) OR $4 = '{}') 
		AND (extension ILIKE ANY($5) OR $5 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $6 OFFSET $7`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{
		search,
		fromYear,
		toYear,
		pq.Array(language),
		pq.Array(extension),
		filters.Limit(),
		filters.Offset(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	books := []*data.Book{}
	for rows.Next() {
		var book data.Book
		err := rows.Scan(
			&totalRecords,
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
			return nil, data.Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

// UpdateBook updates a book record.
func (r *repository) UpdateBook(book *data.Book) error {
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
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&book.Version)
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

// DeleteBook deletes a book record.
func (r *repository) DeleteBook(bookID int64) error {
	if bookID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM books
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, bookID)
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

// AddDownloadForUser adds a download record for a user.
func (r *repository) AddDownloadForUser(userID int64, bookID int64) error {
	query := `
		INSERT INTO users_downloads (user_id, book_id)
		VALUES ($1, $2)`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_downloads_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// RemoveDownloadForUser removes a download record for user.
func (r *repository) RemoveDownloadForUser(userID int64, bookID int64) error {
	if userID < 1 || bookID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_downloads
		WHERE user_id = $1 AND book_id = $2`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, args...)
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

// FavouriteBook favourites a book record.
func (r *repository) FavouriteBook(userID int64, bookID int64) error {
	query := `
		INSERT INTO users_favourite_books (user_id, book_id)
		VALUES ($1, $2)`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_favourite_books_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// DeleteFavouriteBook deletes a favourited book record.
func (r *repository) DeleteFavouriteBook(userID int64, bookID int64) error {
	if userID < 1 || bookID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_favourite_books
		WHERE user_id = $1 AND book_id = $2`
	args := []interface{}{userID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, args...)
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
