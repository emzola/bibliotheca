package data

// Category defines a category.
type Category struct {
	ID         int64  `json:"id"`
	Name       string `json:"category"`
	BooksCount int64  `json:"books_count"`
}
