package dto

import "github.com/emzola/bibliotheca/data"

// CreateBooklistRequestBody defines a request body for CreateBooklist service.
type CreateBooklistRequestBody struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
}

// QsShowBooklist defines the query strings used for ShowBbooklist service.
type QsShowBooklist struct {
	Filters data.Filters
}

// QsListBooklists defines the query strings used for ListBooklists service.
type QsListBooklists struct {
	Search  string
	Filters data.Filters
}

// QsFindBooksForBooklist defines the query strings used for QsFindBooksForBooklist service.
type QsFindBooksForBooklist struct {
	Search  string
	Filters data.Filters
}

// UpdateBooklistRequestBody defines the request body for UpdateBooklist service.
type UpdateBooklistRequestBody struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Private     *bool   `json:"private"`
}
