package data

import "time"

// Book contains the data field and types for a book.
type Book struct {
	ID          int64             `json:"id"`
	UserID      int64             `json:"-"`
	CreatedAt   time.Time         `json:"-"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Author      string            `json:"author,omitempty"`
	Category    string            `json:"category,omitempty"`
	Publisher   string            `json:"publisher,omitempty"`
	Language    string            `json:"language,omitempty"`
	Series      string            `json:"series,omitempty"`
	Volume      int32             `json:"volume,omitempty"`
	Edition     int32             `json:"edition,omitempty"`
	Year        int32             `json:"year,omitempty"`
	PageCount   int32             `json:"pages,omitempty"`
	Isbn10      string            `json:"isbn10,omitempty"`
	Isbn13      string            `json:"isbn13,omitempty"`
	Cover       string            `json:"cover,omitempty"`
	Path        string            `json:"path"`
	Info        map[string]string `json:"info"` // original filename and size (in KB or MB)
	Version     int32             `json:"version"`
}
