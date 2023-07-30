package dto

import "github.com/emzola/bibliotheca/data"

// QsShowCategory defines the query strings used for showing a category.
type QsShowCategory struct {
	Filters data.Filters
}
