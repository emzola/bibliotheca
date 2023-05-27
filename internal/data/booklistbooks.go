package data

// import (
// 	"context"
// 	"database/sql"
// 	"time"
// )

// // The BooklistBook struct contains the data fields for a booklist's book.
// type BooklistBook struct {
// 	BooklistID int64     `json:"booklist_id"`
// 	BookID     int64     `json:"book_id"`
// 	DateTime   time.Time `json:"date_time"`
// }

// // The BooklistBookModel struct wraps a sql.DB connection pool for Booklist's book.
// type BooklistBookModel struct {
// 	DB *sql.DB
// }

// func (m BooklistBookModel) Insert(booklistID, bookID int64) error {
// 	query := `
// 		INSERT INTO booklists_books (booklist_id, book_id)
// 		VALUES ($1, $2)`
// 	args := []interface{}{booklistID, bookID}
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	_, err := m.DB.ExecContext(ctx, query, args...)
// 	return err
// }

// func (m BooklistBookModel) Delete(booklistID, bookID int64) error {
// 	if booklistID < 1 || bookID < 1 {
// 		return ErrRecordNotFound
// 	}
// 	query := `
// 		DELETE FROM booklists_books
// 		WHERE booklist_id = $1 AND book_id = $2`
// 	args := []interface{}{booklistID, bookID}
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	result, err := m.DB.ExecContext(ctx, query, args...)
// 	if err != nil {
// 		return err
// 	}
// 	rowsAffected, err := result.RowsAffected()
// 	if err != nil {
// 		return err
// 	}
// 	if rowsAffected == 0 {
// 		return ErrRecordNotFound
// 	}
// 	return nil
// }
