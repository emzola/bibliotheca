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

type booklists interface {
	CreateBooklist(booklist *data.Booklist) error
	GetBooklist(booklistID int64) (*data.Booklist, error)
	UpdateBooklist(booklist *data.Booklist) error
	DeleteBooklist(booklistID int64) error
	GetAllBooklists(item string, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	FavouriteBooklist(userID, booklistID int64) error
	DeleteFavouriteBooklist(userID int64, booklistID int64) error
	GetAllBooksForBooklist(booklistID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	AddBookToBooklist(booklistID, bookID int64) error
	DeleteBookFromBooklist(booklistID, bookID int64) error
	SearchBooksInBooklist(search string, filters data.Filters) ([]*data.Book, data.Metadata, error)
}

// CreateBooklist creates a new booklist record.
func (r *repository) CreateBooklist(booklist *data.Booklist) error {
	query := `
		INSERT INTO booklists (user_id, name, description, private)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, version`
	args := []interface{}{booklist.UserID, booklist.Name, booklist.Description, booklist.Private}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&booklist.ID, &booklist.CreatedAt, &booklist.UpdatedAt, &booklist.Version)
}

// GetBooklist retrieves a booklist record.
func (r *repository) GetBooklist(booklistID int64) (*data.Booklist, error) {
	if booklistID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT booklists.id, booklists.user_id, users.name, booklists.name, booklists.description, booklists.private, booklists.created_at, booklists.updated_at, booklists.version
		FROM booklists
		INNER JOIN users ON booklists.user_id = users.id
		WHERE booklists.id = $1`
	var booklist data.Booklist
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, booklistID).Scan(
		&booklist.ID,
		&booklist.UserID,
		&booklist.CreatorName,
		&booklist.Name,
		&booklist.Description,
		&booklist.Private,
		&booklist.CreatedAt,
		&booklist.UpdatedAt,
		&booklist.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &booklist, nil
}

// UpdateBooklist updates a booklist record.
func (r *repository) UpdateBooklist(booklist *data.Booklist) error {
	query := `
		UPDATE booklists
		SET name = $1, description = $2, private = $3, updated_at = CURRENT_TIMESTAMP(0), version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING updated_at, version`
	args := []interface{}{booklist.Name, booklist.Description, booklist.Private, booklist.ID, booklist.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&booklist.UpdatedAt, &booklist.Version)
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

// DeleteBooklist deletes a booklist record.
func (r *repository) DeleteBooklist(booklistID int64) error {
	if booklistID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM booklists
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, booklistID)
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

// GetAllBooksForBooklist retrieves all paginated book records for a booklist.
// Records can be filtered and sorted.
func (r *repository) GetAllBooksForBooklist(booklistID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count (*) OVER(), books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
		FROM books
		INNER JOIN booklists_books ON booklists_books.book_id = books.id
		INNER JOIN booklists ON booklists_books.booklist_id = booklists.id
		WHERE booklists.id = $1
		ORDER BY %s %s, datetime DESC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{booklistID, filters.Limit(), filters.Offset()}
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

// GetAllBooklists retrieves a paginated list of all booklist records.
// Records can be filtered and sorted.
func (r *repository) GetAllBooklists(item string, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), booklists.id, booklists.user_id, users.name, booklists.name, booklists.description, booklists.private, booklists.created_at, booklists.updated_at, booklists.version
		FROM booklists  
		INNER JOIN users on booklists.user_id = users.id
		WHERE (
			to_tsvector('simple', booklists.name) || to_tsvector('simple', booklists.description)
			@@ plainto_tsquery('simple', $1) OR $1 = ''
		)
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{
		item,
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
	booklists := []*data.Booklist{}
	for rows.Next() {
		var booklist data.Booklist
		err := rows.Scan(
			&totalRecords,
			&booklist.ID,
			&booklist.UserID,
			&booklist.CreatorName,
			&booklist.Name,
			&booklist.Description,
			&booklist.Private,
			&booklist.CreatedAt,
			&booklist.UpdatedAt,
			&booklist.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}

// FavouriteBooklist favourites a booklist.
func (r *repository) FavouriteBooklist(userID, booklistID int64) error {
	query := `
		INSERT INTO users_favourite_booklists (user_id, booklist_id)
		VALUES ($1, $2)`
	args := []interface{}{userID, booklistID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_favourite_booklists_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// DeleteFavouriteBooklist deletes a fovourited booklist record.
func (r *repository) DeleteFavouriteBooklist(userID int64, booklistID int64) error {
	if userID < 1 || booklistID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_favourite_booklists
		WHERE user_id = $1 AND booklist_id = $2`
	args := []interface{}{userID, booklistID}
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

// AddBookToBooklist adds a book record to a booklist.
func (r *repository) AddBookToBooklist(booklistID, bookID int64) error {
	query := `
		INSERT INTO booklists_books (booklist_id, book_id)
		VALUES ($1, $2)`
	args := []interface{}{booklistID, bookID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "booklists_books_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// RemoveBookFromBooklist deletes a book record from booklist.
func (r *repository) DeleteBookFromBooklist(booklistID, bookID int64) error {
	if booklistID < 1 || bookID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM booklists_books
		WHERE booklist_id = $1 AND book_id = $2`
	args := []interface{}{booklistID, bookID}
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

// SearchBooksInBooklist finds book records inside a booklist.
func (r *repository) SearchBooksInBooklist(search string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
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
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{search, filters.Limit(), filters.Offset()}
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
