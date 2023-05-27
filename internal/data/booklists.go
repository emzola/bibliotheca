package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// The Booklist struct contains the data fields for a booklist.
type Booklist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Username    string    `json:"username"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"-"`
}

func ValidateBooklist(v *validator.Validator, booklist *Booklist) {
	v.Check(booklist.Name != "", "name", "must be provided")
	v.Check(len(booklist.Name) <= 500, "name", "must not be more than 500 bytes long")
	v.Check(len(booklist.Description) <= 1000, "description", "must not be more than 1000 bytes long")
}

// The BooklistModel struct wraps a sql.DB connection pool for Booklist.
type BooklistModel struct {
	DB *sql.DB
}

func (m BooklistModel) Insert(booklist *Booklist) error {
	query := `
		INSERT INTO booklists (user_id, name, description, private)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at, version`
	args := []interface{}{booklist.UserID, booklist.Name, booklist.Description, booklist.Private}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&booklist.ID, &booklist.CreatedAt, &booklist.UpdatedAt, &booklist.Version)
}

func (m BooklistModel) Get(id int64) (*Booklist, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, user_id, name, description, private, created_at, updated_at, version
		FROM booklists
		WHERE id = $1`
	var booklist Booklist
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&booklist.ID,
		&booklist.UserID,
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

func (m BooklistModel) Update(booklist *Booklist) error {
	query := `
		UPDATE booklists
		SET name = $1, description = $2, private = $3, updated_at = CURRENT_TIMESTAMP(0), version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING updated_at, version`
	args := []interface{}{booklist.Name, booklist.Description, booklist.Private, booklist.ID, booklist.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&booklist.UpdatedAt, &booklist.Version)
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

func (m BooklistModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM booklists
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
