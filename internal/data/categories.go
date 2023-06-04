package data

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/net/context"
)

var ErrDuplicateCategory = errors.New("duplicate category")

// The Category struct contains the data fields for a category.
type Category struct {
	ID         int64  `json:"id"`
	Name       string `json:"category"`
	BooksCount int64  `json:"books_count"`
}

type CategoryModel struct {
	DB *sql.DB
}

func (m CategoryModel) Get(id int64) (*Category, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, name
		FROM categories 
		WHERE id = $1`
	var category Category
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
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

func (m CategoryModel) GetAll() ([]*Category, error) {
	query := `
		SELECT categories.id, categories.name, COUNT(books_categories.book_id)
		FROM categories
		LEFT JOIN books_categories ON books_categories.category_id = categories.id
		GROUP BY categories.id 
		ORDER by categories.id ASC`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	categories := []*Category{}
	for rows.Next() {
		var category Category
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

func (m CategoryModel) AddForBook(bookID int64, category string) error {
	query := `
		INSERT INTO books_categories
		SELECT $1, categories.id FROM categories WHERE categories.name = $2`
	args := []interface{}{bookID, category}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "books_categories_pkey"`:
			return ErrDuplicateCategory
		default:
			return err
		}
	}
	return nil
}

func (m CategoryModel) GetForBook(id int64) (*Category, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT categories.id, categories.name
		FROM categories 
		INNER JOIN books_categories ON books_categories.category_id = categories.id
		INNER JOIN books ON books_categories.book_id = books.id
		WHERE books.id = $1`
	var category Category
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
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

func (m CategoryModel) DeleteForBook(bookID, categoryID int64) error {
	query := `
		DELETE FROM books_categories 
		WHERE book_id = $1 AND category_id = $2`
	args := []interface{}{bookID, categoryID}
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
