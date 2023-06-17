package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

var ErrDuplicateBooklistFavourite = errors.New("duplicate booklist favourite")

// The Booklist struct contains the data fields for a booklist.
type Booklist struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"creator_id"`
	CreatorName string    `json:"creator_name"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Private     bool      `json:"private"`
	Content     Books     `json:"content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Version     int32     `json:"-"`
}

// The Books struct contains the books and metadata content of a specific booklist
type Books struct {
	Books    []*Book  `json:"books,omitempty"`
	Metadata Metadata `json:"metadata,omitempty"`
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
		SELECT booklists.id, booklists.user_id, users.name, booklists.name, booklists.description, booklists.private, booklists.created_at, booklists.updated_at, booklists.version
		FROM booklists
		INNER JOIN users ON booklists.user_id = users.id
		WHERE booklists.id = $1`
	var booklist Booklist
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
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

func (m BooklistModel) GetAll(item string, filters Filters) ([]*Booklist, Metadata, error) {
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
		filters.sortColumn(), filters.sortDirection(),
	)
	args := []interface{}{
		item,
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
	booklists := []*Booklist{}
	for rows.Next() {
		var booklist Booklist
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
			return nil, Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}

func (m BooklistModel) AddFavouriteForUser(userID, booklistID int64) error {
	query := `
		INSERT INTO users_favourite_booklists (user_id, booklist_id)
		VALUES ($1, $2)`
	args := []interface{}{userID, booklistID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_favourite_booklists_pkey"`:
			return ErrDuplicateBooklistFavourite
		default:
			return err
		}
	}
	return nil
}

func (m BooklistModel) RemoveFavouriteForUser(userID, booklistID int64) error {
	if userID < 1 || booklistID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_favourite_booklists
		WHERE user_id = $1 AND booklist_id = $2`
	args := []interface{}{userID, booklistID}
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

func (m BooklistModel) GetAllFavouritesForUser(userID int64, filters Filters) ([]*Booklist, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), booklists.id, booklists.user_id, booklists.name, booklists.description, booklists.private, booklists.created_at, booklists.updated_at, booklists.version
		FROM booklists
		INNER JOIN users_favourite_booklists ON users_favourite_booklists.booklist_id = booklists.id
		INNER JOIN users ON users_favourite_booklists.user_id = users.id
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
	totalRecords := 0
	booklists := []*Booklist{}
	for rows.Next() {
		var booklist Booklist
		err := rows.Scan(
			&totalRecords,
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
			return nil, Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}

func (m BooklistModel) GetAllForUser(userID int64, filters Filters) ([]*Booklist, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, name, description, private, created_at, updated_at, version
		FROM booklists
		WHERE user_id = $1
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
	totalRecords := 0
	booklists := []*Booklist{}
	for rows.Next() {
		var booklist Booklist
		err := rows.Scan(
			&totalRecords,
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
			return nil, Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}
