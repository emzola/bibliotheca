package service

import (
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/emzola/bibliotheca/clients"
	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type books interface {
	CreateBook(userID int64, r *http.Request) (*data.Book, error)
	GetBook(bookID int64) (*data.Book, error)
	ListBooks(search string, fromYear int, toYear int, language []string, extension []string, filters data.Filters) ([]*data.Book, data.Metadata, error)
	UpdateBook(bookID int64, requestBody dto.UpdateBookRequestBody) (*data.Book, error)
	UpdateBookCover(bookID int64, r *http.Request) (*data.Book, error)
	DeleteBook(bookID int64) error
	DownloadBook(bookID int64, userID int64) error
	DeleteBookFromDownloads(userID int64, bookID int64) error
	FavouriteBook(userID int64, bookID int64) error
	DeleteFavouriteBook(userID int64, bookID int64) error
}

// CreateBook service creates a new book.
func (s *service) CreateBook(userID int64, r *http.Request) (*data.Book, error) {
	// Parse form data
	err := r.ParseMultipartForm(5000)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesError):
			return nil, ErrContentTooLarge
		default:
			return nil, ErrBadRequest
		}
	}
	file, fileHeader, err := r.FormFile("book")
	if err != nil {
		return nil, ErrBadRequest
	}
	defer file.Close()
	// Detect file Mime type
	buffer, mtype, err := s.detectMimeType(file, fileHeader)
	if err != nil {
		return nil, err
	}
	// Check whether Mime type is supported
	supportedMediaType := []string{
		"application/pdf",
		"application/epub+zip",
		"application/x-ms-reader",
		"application/x-mobipocket-ebook",
		"application/vnd.oasis.opendocument.text",
		"text/rtf",
		"image/vnd.djvu",
	}
	if validMime := validator.Mime(mtype, supportedMediaType...); !validMime {
		return nil, ErrUnsupportedMediaType
	}
	// Upload file to s3 object storage
	s3Client, err := clients.NewS3Client(s.config)
	if err != nil {
		return nil, err
	}
	s3FileKey, err := s.uploadFileToS3(s3Client, buffer, mtype, fileHeader, data.ScopeBook)
	if err != nil {
		return nil, err
	}
	book := &data.Book{
		UserID:    userID,
		Title:     strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename)),
		S3FileKey: s3FileKey,
		Filename:  fileHeader.Filename,
		Extension: strings.ToUpper(strings.TrimPrefix(filepath.Ext(fileHeader.Filename), ".")),
		Size:      fileHeader.Size,
	}
	// Create record
	err = s.repo.CreateBook(book)
	if err != nil {
		return nil, err
	}
	return book, nil
}

// ShowBook service retrieves the details of a book.
func (s *service) GetBook(bookID int64) (*data.Book, error) {
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return book, nil
}

// ListBooks service retrieves a list of paginated books. The list can be filtered and sorted.
func (s *service) ListBooks(search string, fromYear int, toYear int, language []string, extension []string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.GetAllBooks(search, fromYear, toYear, language, extension, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}

// UpdateBook service updates the details of a specific book.
func (s *service) UpdateBook(bookID int64, requestBody dto.UpdateBookRequestBody) (*data.Book, error) {
	// Retrieve the book by its ID
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Update only fields with new data
	if requestBody.Title != nil {
		book.Title = *requestBody.Title
	}
	if requestBody.Description != nil {
		book.Description = *requestBody.Description
	}
	if requestBody.Author != nil {
		book.Author = requestBody.Author
	}
	if requestBody.Category != nil {
		book.Category = *requestBody.Category
	}
	if requestBody.Publisher != nil {
		book.Publisher = *requestBody.Publisher
	}
	if requestBody.Language != nil {
		book.Language = *requestBody.Language
	}
	if requestBody.Series != nil {
		book.Series = *requestBody.Series
	}
	if requestBody.Volume != nil {
		book.Volume = *requestBody.Volume
	}
	if requestBody.Edition != nil {
		book.Edition = *requestBody.Edition
	}
	if requestBody.Year != nil {
		book.Year = *requestBody.Year
	}
	if requestBody.PageCount != nil {
		book.PageCount = *requestBody.PageCount
	}
	if requestBody.Isbn10 != nil {
		book.Isbn10 = *requestBody.Isbn10
	}
	if requestBody.Isbn13 != nil {
		book.Isbn13 = *requestBody.Isbn13
	}
	if requestBody.Popularity != nil {
		book.Popularity = *requestBody.Popularity
	}
	v := validator.New()
	if data.ValidateBook(v, book); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.UpdateBook(book)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	// Also update books_categories table to detect the right book associated with a category
	if requestBody.Category != nil {
		// First check if a category for a book exists in books_categories table
		category, err := s.repo.GetCategoryForBook(book.ID)
		if err != nil {
			switch {
			// If ErrRecordNotFound is returned, it means no category was found,
			// so add a category for the book in the books_categories table
			case errors.Is(err, repository.ErrRecordNotFound):
				err := s.repo.AddCategoryForBook(book.ID, book.Category)
				if err != nil {
					switch {
					case errors.Is(err, repository.ErrDuplicateRecord):
						return nil, ErrDuplicateRecord
					default:
						return nil, err
					}
				}
			default:
				return nil, err
			}
		} else {
			// At this point, a category already exists in the books_categories table.
			// So, delete the record and add a new record with updated info
			err := s.repo.DeleteCategoryForBook(book.ID, category.ID)
			if err != nil {
				switch {
				case errors.Is(err, repository.ErrRecordNotFound):
					return nil, ErrRecordNotFound
				default:
					return nil, err
				}
			}
			err = s.repo.AddCategoryForBook(book.ID, book.Category)
			if err != nil {
				switch {
				case errors.Is(err, repository.ErrDuplicateRecord):
					return nil, ErrDuplicateRecord
				default:
					return nil, err
				}
			}
		}
	}
	return book, nil
}

// UpdateBookCover service uploads a cover image for a book.
func (s *service) UpdateBookCover(bookID int64, r *http.Request) (*data.Book, error) {
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	err = r.ParseMultipartForm(5000)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesError):
			return nil, ErrContentTooLarge
		default:
			return nil, ErrBadRequest
		}
	}
	file, fileHeader, err := r.FormFile("cover")
	if err != nil {
		return nil, ErrBadRequest
	}
	// Detect image Mime type
	buffer, mtype, err := s.detectMimeType(file, fileHeader)
	if err != nil {
		return nil, err
	}
	// Check whether Mime type is supported
	supportedMediaType := []string{
		"image/jpeg",
		"image/png",
	}
	if v := validator.Mime(mtype, supportedMediaType...); !v {
		return nil, ErrUnsupportedMediaType
	}
	// Upload image to S3 object storage
	s3Client, err := clients.NewS3Client(s.config)
	if err != nil {
		return nil, err
	}
	s3CoverPath, err := s.uploadFileToS3(s3Client, buffer, mtype, fileHeader, data.ScopeCover)
	if err != nil {
		return nil, err
	}
	book.CoverPath = s3CoverPath
	// Update book record
	err = s.repo.UpdateBook(book)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	return book, nil
}

// DeleteBook service deletes a book.
func (s *service) DeleteBook(bookID int64) error {
	err := s.repo.DeleteBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}

// DownloadBook service downloads a book.
func (s *service) DownloadBook(bookID int64, userID int64) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	// Check user's daily download limit. If it exceeds daily limit, do nothing further
	if user.DownloadCount >= data.DailyDownloadLimit {
		return ErrNotPermitted
	}
	// Otherwise, proceed with other actions as normal
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	s3Client, err := clients.NewS3Client(s.config)
	if err != nil {
		return err
	}
	// Download book in a background goroutine to speed up response time
	s.background(func() {
		err := s.downloadFileFromS3(s3Client, book)
		if err != nil {
			s.logger.PrintError(err, nil)
			return
		}
	})
	// Add record to downloads table
	err = s.repo.AddDownloadForUser(user.ID, book.ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			err := s.repo.RemoveDownloadForUser(user.ID, book.ID)
			if err != nil {
				switch {
				case errors.Is(err, repository.ErrRecordNotFound):
					return ErrRecordNotFound
				default:
					return err
				}
			}
			err = s.repo.AddDownloadForUser(user.ID, book.ID)
			if err != nil {
				return err
			}
		default:
			return err
		}
	}
	// Increase user download count
	user.DownloadCount++
	err = s.repo.UpdateUser(user)
	if err != nil {
		return err
	}
	return nil
}

// DeleteBookFromDownloads service deletes a book from user's download history.
func (s *service) DeleteBookFromDownloads(userID int64, bookID int64) error {
	err := s.repo.RemoveDownloadForUser(userID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}

// FavouriteBook service marks a book as favourite.
func (s *service) FavouriteBook(userID int64, bookID int64) error {
	err := s.repo.FavouriteBook(userID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// DeleteFavouriteBook service unmarks a book as favourite.
func (s *service) DeleteFavouriteBook(userID int64, bookID int64) error {
	err := s.repo.DeleteFavouriteBook(userID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}
