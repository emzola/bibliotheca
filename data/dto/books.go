package dto

import "github.com/emzola/bibliotheca/data"

// QsListBooks defines the query strings used for listing books.
type QsListBooks struct {
	Search    string
	FromYear  int
	ToYear    int
	Language  []string
	Extension []string
	Filters   data.Filters
}

// UpdateBookRequestBody defines the request body for UpdateBook service. The fields are set
// to a pointer type to allow partial updates based on whether the value if set to nil.
type UpdateBookRequestBody struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Author      []string `json:"author"`
	Category    *string  `json:"category"`
	Publisher   *string  `json:"publisher"`
	Language    *string  `json:"language"`
	Series      *string  `json:"series"`
	Volume      *int32   `json:"volume"`
	Edition     *string  `json:"edition"`
	Year        *int32   `json:"year"`
	PageCount   *int32   `json:"page_count"`
	Isbn10      *string  `json:"isbn_10"`
	Isbn13      *string  `json:"isbn_13"`
	Popularity  *float64 `json:"popularity"`
}

// QsListUserBooks defines query strings for ListUserBooks service.
type QsListUserBooks struct {
	Filters data.Filters
}
