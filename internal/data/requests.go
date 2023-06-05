package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
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
	ID        int64     `json:"id,omitempty"`
	UserID    int64     `json:"user_id,omitempty"`
	Title     string    `json:"title,omitempty"`
	Author    []string  `json:"author,omitempty"`
	Publisher string    `json:"publisher,omitempty"`
	Isbn      string    `json:"isbn,omitempty"`
	Year      int32     `json:"year,omitempty"`
	Language  string    `json:"language,omitempty"`
	Waitlist  int32     `json:"waitlist,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// The RequestModel struct wraps a sql.DB connection pool for Request.
type RequestModel struct {
	DB *sql.DB
}

func (m RequestModel) Insert(request *Request) error {
	query := `
		INSERT INTO requests (user_id, title, author, publisher, isbn, year, language)	
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	  	RETURNING id, created_at`
	args := []interface{}{request.UserID, request.Title, pq.Array(request.Author), request.Publisher, request.Isbn, request.Year, request.Language}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&request.ID, &request.CreatedAt)
}
