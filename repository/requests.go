package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/bibliotheca/data"
)

type requests interface {
	CreateRequest(request *data.Request) error
	UpdateRequest(request *data.Request) error
	GetRequest(requestID int64) (*data.Request, error)
	GetAllRequests(search, status string, filters data.Filters) ([]*data.Request, data.Metadata, error)
	AddRequestForUser(userID, requestID int64, expiry time.Time) error
	DeleteRequestForUser(userID int64, requestID int64) error
}

// CreateRequest creates a new book request record.
func (r *repository) CreateRequest(request *data.Request) error {
	query := `
		INSERT INTO requests (user_id, title, publisher, isbn, year, expiry, status)	
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	  	RETURNING id, created_at, version`
	args := []interface{}{
		request.UserID,
		request.Title,
		request.Publisher,
		request.Isbn,
		request.Year,
		request.Expiry,
		request.Status,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&request.ID, &request.CreatedAt, &request.Version)
}

// AddRequestForUser adds a book request subscribe record for user.
func (r *repository) AddRequestForUser(userID, requestID int64, expiry time.Time) error {
	query := `
		INSERT INTO users_requests (user_id, request_id, expiry)
		VALUES ($1, $2, $3)`
	args := []interface{}{userID, requestID, expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_requests_pkey"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// DeleteRequestForUser deletes a request subscription record for user.
func (r *repository) DeleteRequestForUser(userID int64, requestID int64) error {
	if userID < 1 || requestID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_requests
		WHERE user_id = $1 AND request_id = $2`
	args := []interface{}{userID, requestID}
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

// UpdateRequest updates a request record.
func (r *repository) UpdateRequest(request *data.Request) error {
	query := `
		UPDATE requests
		SET title = $1, publisher = $2, isbn = $3, year = $4, expiry = $5, status = $6, waitlist = $7, version = version + 1
		WHERE id = $8 AND version = $9
		RETURNING version`
	args := []interface{}{
		request.Title,
		request.Publisher,
		request.Isbn,
		request.Year,
		request.Expiry,
		request.Status,
		request.Waitlist,
		request.ID,
		request.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&request.Version)
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

// GetRequest retrieves a request record.
func (r *repository) GetRequest(requestID int64) (*data.Request, error) {
	if requestID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, user_id, title, publisher, isbn, year, expiry, status, waitlist, created_at, version
		FROM requests 
		WHERE id = $1`
	var request data.Request
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, requestID).Scan(
		&request.ID,
		&request.UserID,
		&request.Title,
		&request.Publisher,
		&request.Isbn,
		&request.Year,
		&request.Expiry,
		&request.Status,
		&request.Waitlist,
		&request.CreatedAt,
		&request.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &request, nil
}

// GetAllRequests retrieves a paginated list of all request records.
// Records can be filtered and sorted.
func (r *repository) GetAllRequests(search, status string, filters data.Filters) ([]*data.Request, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, title, publisher, isbn, year, expiry, status, waitlist, created_at, version
		FROM requests
		WHERE (
			to_tsvector('simple', title) || 
			to_tsvector('simple', isbn) || 
			to_tsvector('simple', publisher) 
			@@ plainto_tsquery('simple', $1) OR $1 = ''
		) 
		AND (LOWER(status) = LOWER($2) OR $2 = '') 
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{
		search,
		status,
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
	requests := []*data.Request{}
	for rows.Next() {
		var request data.Request
		err := rows.Scan(
			&totalRecords,
			&request.ID,
			&request.UserID,
			&request.Title,
			&request.Publisher,
			&request.Isbn,
			&request.Year,
			&request.Expiry,
			&request.Status,
			&request.Waitlist,
			&request.CreatedAt,
			&request.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		requests = append(requests, &request)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return requests, metadata, nil
}
