package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emzola/bibliotheca/data"
	"github.com/lib/pq"
)

type users interface {
	RegisterUser(user *data.User) error
	GetUserByID(ID int64) (*data.User, error)
	GetUserByEmail(email string) (*data.User, error)
	UpdateUser(user *data.User) error
	DeleteUser(ID int64) error
	GetUserForToken(tokenScope string, tokenPlaintext string) (*data.User, error)
	GetAllFavouriteBooklistsForUser(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	GetAllBooklistsForUser(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	GetAllRequestsForUser(userID int64, status string, filters data.Filters) ([]*data.Request, data.Metadata, error)
	GetAllBooksForUser(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	GetAllFavouriteBooksForUser(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	GetAllDownloadsForUser(userID int64, fromDate, toDate string, filters data.Filters) ([]*data.Book, data.Metadata, error)
}

// RegisterUser registers a new user.
func (r *repository) RegisterUser(user *data.User) error {
	query := `
		INSERT INTO users (name, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []interface{}{user.Name, user.Email, user.Password.Hash, user.Activated}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Version,
	)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// GetUserByID retrieves a user record by its ID.
func (r *repository) GetUserByID(ID int64) (*data.User, error) {
	query := `
		SELECT id, created_at, name, email, password_hash, activated, download_count, version
		FROM users
		WHERE id = $1`
	var user data.User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, ID).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.DownloadCount,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

// GetUserByID retrieves a user record by its email.
func (r *repository) GetUserByEmail(email string) (*data.User, error) {
	query := `
		SELECT id, created_at, name, email, password_hash, activated, download_count, version
		FROM users
		WHERE email = $1`
	var user data.User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.DownloadCount,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

// UpdateUser updates a user record.
func (r *repository) UpdateUser(user *data.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, activated = $4, download_count = $5, version = version + 1
		WHERE id = $6 AND version = $7
		RETURNING version`
	args := []interface{}{
		user.Name,
		user.Email,
		user.Password.Hash,
		user.Activated,
		user.DownloadCount,
		user.ID,
		user.Version,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&user.Version)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateRecord
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

// DeleteUser deletes a user record.
func (r *repository) DeleteUser(ID int64) error {
	if ID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM users
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, ID)
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

// GetUserForToken returns a user record associated with a token.
func (r *repository) GetUserForToken(tokenScope string, tokenPlaintext string) (*data.User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))
	query := `
		SELECT users.id, users.created_at, users.name, users.email, users.password_hash, users.activated, users.version
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3`
	args := []interface{}{tokenHash[:], tokenScope, time.Now()}
	var user data.User
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &user, nil
}

// GetAllFavouriteBooklistsForUser retrieves a paginated record of all user favourite booklist.
// Records can be filtered and sorted.
func (r *repository) GetAllFavouriteBooklistsForUser(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), booklists.id, booklists.user_id, booklists.name, booklists.description, booklists.private, booklists.created_at, booklists.updated_at, booklists.version
		FROM booklists
		INNER JOIN users_favourite_booklists ON users_favourite_booklists.booklist_id = booklists.id
		INNER JOIN users ON users_favourite_booklists.user_id = users.id
		WHERE users.id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	booklists := []*data.Booklist{}
	for rows.Next() {
		var booklist data.Booklist
		err := rows.Scan(
			&totalRecords,
			&booklist.ID,
			&booklist.UserID,
			&booklist.Name,
			&booklist.Description,
			&booklist.Private,
			&booklist.CreatedAt,
			&booklist.UpdatedAt,
			&booklist.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}

// GetAllBooklistsForUser retrieves a paginated record of a user's booklist.
// Records can be filtered and sorted
func (r *repository) GetAllBooklistsForUser(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, name, description, private, created_at, updated_at, version
		FROM booklists
		WHERE user_id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	booklists := []*data.Booklist{}
	for rows.Next() {
		var booklist data.Booklist
		err := rows.Scan(
			&totalRecords,
			&booklist.ID,
			&booklist.UserID,
			&booklist.Name,
			&booklist.Description,
			&booklist.Private,
			&booklist.CreatedAt,
			&booklist.UpdatedAt,
			&booklist.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		booklists = append(booklists, &booklist)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return booklists, metadata, nil
}

// GetAllRequestsForUser retrieves a paginated record of all user's requests.
// Records can be filtered and sorted.
func (r *repository) GetAllRequestsForUser(userID int64, status string, filters data.Filters) ([]*data.Request, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), requests.id, requests.user_id, requests.title, requests.publisher, requests.isbn, requests.year, requests.expiry, requests.status, requests.waitlist, requests.created_at, requests.version
		FROM requests
		INNER JOIN users_requests ON users_requests.request_id = requests.id
		INNER JOIN users ON users_requests.user_id = users.id
		WHERE users.id = $1 AND (LOWER(requests.status) = LOWER($2) OR $2 = '') 
		ORDER BY %s %s, datetime DESC
		LIMIT $3 OFFSET $4`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, status, filters.Limit(), filters.Offset()}
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

// GetAllBooksForUser retrieves all book records for a user.
// Records can be filtered and sorted.
func (r *repository) GetAllBooksForUser(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, user_id, created_at, title, description, author, category, publisher, language, series, volume, edition, year, page_count, isbn_10, isbn_13, cover_path, s3_file_key, fname, extension, size, popularity, version
		FROM books  
		WHERE user_id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	books := []*data.Book{}
	for rows.Next() {
		var book data.Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.UserID,
			&book.CreatedAt,
			&book.Title,
			&book.Description,
			pq.Array(&book.Author),
			&book.Category,
			&book.Publisher,
			&book.Language,
			&book.Series,
			&book.Volume,
			&book.Edition,
			&book.Year,
			&book.PageCount,
			&book.Isbn10,
			&book.Isbn13,
			&book.CoverPath,
			&book.S3FileKey,
			&book.Filename,
			&book.Extension,
			&book.Size,
			&book.Popularity,
			&book.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

// GetAllFavouriteBooksForUser retrieves a paginated record of user's favourite books.
// Records can be filtered and sorted.
func (r *repository) GetAllFavouriteBooksForUser(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
		FROM books
		INNER JOIN users_favourite_books ON users_favourite_books.book_id = books.id
		INNER JOIN users ON users_favourite_books.user_id = users.id
		WHERE users.id = $1
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	defer rows.Close()
	totalRecords := 0
	books := []*data.Book{}
	for rows.Next() {
		var book data.Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.UserID,
			&book.CreatedAt,
			&book.Title,
			&book.Description,
			pq.Array(book.Author),
			&book.Category,
			&book.Publisher,
			&book.Language,
			&book.Series,
			&book.Volume,
			&book.Edition,
			&book.Year,
			&book.PageCount,
			&book.Isbn10,
			&book.Isbn13,
			&book.CoverPath,
			&book.S3FileKey,
			&book.Filename,
			&book.Extension,
			&book.Size,
			&book.Popularity,
			&book.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

// GetAllDownloadsForUser retrieves all download records for a user.
// Records can be filtered and sorted.
func (r *repository) GetAllDownloadsForUser(userID int64, fromDate, toDate string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
		FROM books
		INNER JOIN users_downloads ON users_downloads.book_id = books.id
		INNER JOIN users ON users_downloads.user_id = users.id
		WHERE users.id = $1 AND DATE(datetime) BETWEEN TO_DATE($2, 'YYYY-MM-DD') AND TO_DATE($3, 'YYYY-MM-DD')
		ORDER BY %s %s, datetime DESC
		LIMIT $4 OFFSET $5`,
		filters.SortColumn(), filters.SortDirection(),
	)
	args := []interface{}{userID, fromDate, toDate, filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	defer rows.Close()
	totalRecords := 0
	books := []*data.Book{}
	for rows.Next() {
		var book data.Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.UserID,
			&book.CreatedAt,
			&book.Title,
			&book.Description,
			pq.Array(&book.Author),
			&book.Category,
			&book.Publisher,
			&book.Language,
			&book.Series,
			&book.Volume,
			&book.Edition,
			&book.Year,
			&book.PageCount,
			&book.Isbn10,
			&book.Isbn13,
			&book.CoverPath,
			&book.S3FileKey,
			&book.Filename,
			&book.Extension,
			&book.Size,
			&book.Popularity,
			&book.Version,
		)
		if err != nil {
			return nil, data.Metadata{}, err
		}
		books = append(books, &book)
	}
	if err = rows.Err(); err != nil {
		return nil, data.Metadata{}, err
	}
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

// func (r *repository) GetDownloadForUser(userID, bookID int64) (*data.Book, error) {
// 	query := `
// 		SELECT books.id, books.user_id, books.created_at, books.title, books.description, books.author, books.category, books.publisher, books.language, books.series, books.volume, books.edition, books.year, books.page_count, books.isbn_10, books.isbn_13, books.cover_path, books.s3_file_key, books.fname, books.extension, books.size, books.popularity, books.version
// 		FROM books
// 		INNER JOIN users_downloads ON users_downloads.book_id = books.id
// 		INNER JOIN users ON users_downloads.user_id = users.id
// 		WHERE users.id = $1 AND books.id = $2`
// 	var book data.Book
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()
// 	err := r.db.QueryRowContext(ctx, query, userID, bookID).Scan(
// 		&book.ID,
// 		&book.UserID,
// 		&book.CreatedAt,
// 		&book.Title,
// 		&book.Description,
// 		pq.Array(&book.Author),
// 		&book.Category,
// 		&book.Publisher,
// 		&book.Language,
// 		&book.Series,
// 		&book.Volume,
// 		&book.Edition,
// 		&book.Year,
// 		&book.PageCount,
// 		&book.Isbn10,
// 		&book.Isbn13,
// 		&book.CoverPath,
// 		&book.S3FileKey,
// 		&book.Filename,
// 		&book.Extension,
// 		&book.Size,
// 		&book.Popularity,
// 		&book.Version,
// 	)
// 	if err != nil {
// 		switch {
// 		case errors.Is(err, sql.ErrNoRows):
// 			return nil, ErrRecordNotFound
// 		default:
// 			return nil, err
// 		}
// 	}
// 	return &book, nil
// }
