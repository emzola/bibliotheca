package dto

import "github.com/emzola/bibliotheca/data"

// CreateRequestRequestBody defines a request body for CreateRequest service.
type CreateRequestRequestBody struct {
	Isbn string `json:"isbn"`
}

// The OpenLibAPIRequestBody struct contains the expected JSON data that has
// been decoded into a Go type from the openlibrary API.
type OpenLibAPIRequestBody struct {
	Title     string   `json:"title"`
	Publisher []string `json:"publishers"`
	Isbn10    []string `json:"isbn_10"`
	Isbn13    []string `json:"isbn_13"`
	Date      string   `json:"publish_date"`
}

// QsListRequest defines query strings for ListRequest service.
type QsListRequest struct {
	Search  string
	Status  string
	Filters data.Filters
}
