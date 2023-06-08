package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrDuplicateRequest = errors.New("duplicate request")
)

// The BookJSONData struct contains the expected JSON data that has
// been decoded into a Go type from the openlibrary API.
type BookJSONData struct {
	Title  string `json:"title"`
	Author []struct {
		Key string
	} `json:"authors"`
	Publisher []string `json:"publishers"`
	Isbn10    []string `json:"isbn_10"`
	Isbn13    []string `json:"isbn_13"`
	Date      string   `json:"publish_date"`
	Language  []struct {
		Key string
	} `json:"languages"`
}

// The Author struct contains the expected JSON data for an author
// that has been decoded into a Go type from the openlibrary API.
type Author struct {
	Name string `json:"name"`
}

// The Language struct contains the expected JSON data for a language
// that has been decoded into a Go type from the openlibrary API.
type Language struct {
	Name string `json:"name"`
}

// The Request struct contains the data fields for a Request.
type Request struct {
	ID                 int64     `json:"id,omitempty"`
	UserID             int64     `json:"user_id,omitempty"`
	Title              string    `json:"title,omitempty"`
	Author             []string  `json:"author,omitempty"`
	Publisher          string    `json:"publisher,omitempty"`
	Isbn               string    `json:"isbn,omitempty"`
	Year               int32     `json:"year,omitempty"`
	Language           string    `json:"language,omitempty"`
	Expiry             time.Time `json:"expiry,omitempty"`
	Status             string    `json:"status,omitempty"`
	CreatedAt          time.Time `json:"created_at,omitempty"`
	Waitlist           int32     `json:"waitlist,omitempty"`
	SubscriptionExpiry time.Time `json:"subscription_expiry,omitempty"`
	Version            int32     `json:"-"`
}

// The RequestModel struct wraps a sql.DB connection pool for Request.
type RequestModel struct {
	DB *sql.DB
}

func (m RequestModel) Insert(request *Request) error {
	query := `
		INSERT INTO requests (user_id, title, author, publisher, isbn, year, language, expiry, status)	
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	  	RETURNING id, created_at, version`
	args := []interface{}{
		request.UserID,
		request.Title,
		pq.Array(request.Author),
		request.Publisher,
		request.Isbn,
		request.Year,
		request.Language,
		request.Expiry,
		request.Status,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&request.ID, &request.CreatedAt, &request.Version)
}

func (m RequestModel) Get(id int64) (*Request, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, user_id, title, author, publisher, isbn, year, language, expiry, status, created_at, version
		FROM requests 
		WHERE id = $1`
	var request Request
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&request.ID,
		&request.UserID,
		&request.Title,
		pq.Array(request.Author),
		&request.Publisher,
		&request.Isbn,
		&request.Year,
		&request.Language,
		&request.Expiry,
		&request.Status,
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

func (m RequestModel) AddForUser(userID, requestID int64, expiry time.Time) error {
	query := `
		INSERT INTO users_requests (user_id, request_id, expiry)
		VALUES ($1, $2, $3)`
	args := []interface{}{userID, requestID, expiry}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_requests_pkey"`:
			return ErrDuplicateRequest
		default:
			return err
		}
	}
	return nil
}

func (m RequestModel) DeleteForUser(userID, requestID int64) error {
	if userID < 1 || requestID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users_requests
		WHERE user_id = $1 AND request_id = $2`
	args := []interface{}{userID, requestID}
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

// func (m RequestModel) GetWaitlist(requestID int32) (int32, error) {
// 	query := `
// 		SELECT Count(*)
// 		FROM users_requests
// 		WHERE request_id = $1`
// 	var waitlist int32
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	err := m.DB.QueryRowContext(ctx, query, requestID).Scan(
// 		&waitlist,
// 	)
// 	if err != nil {
// 		return 0, err
// 	}
// 	return waitlist, nil
// }

func (m RequestModel) Update(request *Request) error {
	query := `
		UPDATE requests
		SET title = $1, author = $2, publisher = $3, isbn = $4, year = $5, language = $6, expiry = $7, status = $8, version = version + 1
		WHERE id = $9 AND version = $10
		RETURNING version`
	args := []interface{}{
		request.Title,
		pq.Array(request.Author),
		request.Publisher,
		request.Isbn,
		request.Year,
		request.Language,
		request.Expiry,
		request.Status,
		request.ID,
		request.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&request.Version)
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

func (m RequestModel) GetAllForUser(userID int64, status string, filters Filters) ([]*Request, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), requests.id, requests.user_id, requests.title, requests.author, requests.publisher, 
		requests.isbn, requests.year, requests.language, requests.expiry, requests.status, 
		requests.created_at, requests.version, COUNT(users_requests.user_id), users_requests.expiry
		FROM requests
		LEFT JOIN users_requests ON users_requests.request_id = requests.id
		LEFT JOIN users ON users_requests.user_id = users.id
		WHERE users.id = $1 AND (LOWER(requests.status) = LOWER($2) OR $2 = '') 
		GROUP BY requests.id, requests.user_id, requests.title, requests.author, requests.publisher, 
		requests.isbn, requests.year, requests.language, requests.expiry, requests.status, 
		requests.created_at, requests.version, users_requests.expiry, users_requests.datetime 
		ORDER BY %s %s, datetime DESC
		LIMIT $3 OFFSET $4`,
		filters.sortColumn(), filters.sortDirection(),
	)
	args := []interface{}{userID, status, filters.limit(), filters.offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	requests := []*Request{}
	for rows.Next() {
		var request Request
		err := rows.Scan(
			&totalRecords,
			&request.ID,
			&request.UserID,
			&request.Title,
			pq.Array(request.Author),
			&request.Publisher,
			&request.Isbn,
			&request.Year,
			&request.Language,
			&request.Expiry,
			&request.Status,
			&request.CreatedAt,
			&request.Version,
			&request.Waitlist,
			&request.SubscriptionExpiry,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		requests = append(requests, &request)
	}
	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return requests, metadata, nil
}
