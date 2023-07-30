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

type categories interface {
	GetCategory(categoryID int64) (*data.Category, error)
	GetAllCategories() ([]*data.Category, error)
	GetCategoryForBook(ID int64) (*data.Category, error)
	AddCategoryForBook(bookID int64, category string) error
	DeleteCategoryForBook(bookID, categoryID int64) error
	GetAllBooksForCategory(categoryID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
}

// GetCategory retrieves a category record.
func (r *repository) GetCategory(categoryID int64) (*data.Category, error) {
	if categoryID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, name
		FROM categories
		WHERE id = $1`
	var category data.Category
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, categoryID).Scan(
		&category.ID,
		&category.Name,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &category, nil
}

// GetAllCategories retrieves all category records.
func (r *repository) GetAllCategories() ([]*data.Category, error) {
	query := `
		SELECT categories.id, categories.name, count(books_categories.book_id)
		FROM categories
		LEFT JOIN books_categories ON books_categories.category_id = categories.id
		GROUP BY categories.id
		ORDER by categories.id ASC`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := []*data.Category{}
	for rows.Next() {
		var category data.Category
		err := rows.Scan(
			&category.ID,
			&category.Name,
			&category.BooksCount,
		)
		if err != nil {
			return nil, err
		}
		categories = append(categories, &category)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return categories, nil
}

// AddCategoryForBook associates a book record with a category.
func (r *repository) AddCategoryForBook(bookID int64, category string) error {
	query := `
		INSERT INTO books_categories
		SELECT $1, categories.id
		FROM categories
		WHERE categories.name = $2`
	args := []interface{}{bookID, category}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "books_categories_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// GetCategoryForBook retrieves the category associated with a book record.
func (r *repository) GetCategoryForBook(bookID int64) (*data.Category, error) {
	if bookID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT categories.id, categories.name
		FROM categories 
		INNER JOIN books_categories ON books_categories.category_id = categories.id
		INNER JOIN books ON books_categories.book_id = books.id
		WHERE books.id = $1`
	var category data.Category
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, bookID).Scan(
		&category.ID,
		&category.Name,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &category, nil
}

// DeleteCategoryForBook deletes a category assciated with a book record.
func (r *repository) DeleteCategoryForBook(bookID, categoryID int64) error {
	query := `
		DELETE FROM books_categories
		WHERE book_id = $1 AND category_id = $2`
	args := []interface{}{bookID, categoryID}
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

// GetAllBooksForCategory retrieves a paginated record of all books for a specific category.
func (r *repository) GetAllBooksForCategory(categoryID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count (*) OVER(), books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
		FROM books
		INNER JOIN books_categories ON books_categories.book_id = books.id
		INNER JOIN categories ON books_categories.category_id = categories.id
		WHERE categories.id = $1
		ORDER BY %s %s, datetime DESC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{categoryID, filters.Limit(), filters.Offset()}
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
