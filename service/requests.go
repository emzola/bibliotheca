package service

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/emzola/bibliotheca/clients"
	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type requests interface {
	CreateRequest(userID int64, isbn string) (*data.Request, error)
	GetRequest(requestID int64) (*data.Request, error)
	ListRequests(search string, status string, filters data.Filters) ([]*data.Request, data.Metadata, error)
	SubscribeRequest(userID int64, requestID int64) error
	UnsubscribeRequest(userID int64, requestID int64) error
}

func (s *service) CreateRequest(userID int64, isbn string) (*data.Request, error) {
	v := validator.New()
	if data.ValidateRequestIsbn(v, isbn); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	// Fetch JSON data for a book from openlibrary api
	openLibAPI := &dto.OpenLibAPIRequestBody{}
	url := "https://openlibrary.org/isbn/" + isbn + ".json"
	client := clients.NewHTTPClient()
	bytes, err := s.fetchRemoteResource(client, url)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &openLibAPI)
	if err != nil {
		return nil, ErrBadRequest
	}
	year, err := strconv.Atoi(strings.TrimSpace(strings.Split(openLibAPI.Date, ",")[1]))
	if err != nil {
		return nil, err
	}
	request := &data.Request{
		UserID:    userID,
		Title:     openLibAPI.Title,
		Publisher: openLibAPI.Publisher[0],
		Isbn:      isbn,
		Year:      int32(year),
		Expiry:    time.Now().Add(time.Hour * 24 * 182),
		Status:    "active",
	}
	err = s.repo.CreateRequest(request)
	if err != nil {
		return nil, err
	}
	// Add the new request to the users_requests table
	err = s.repo.AddRequestForUser(userID, request.ID, request.Expiry)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			return nil, ErrDuplicateRecord
		default:
			return nil, err
		}
	}
	// Update request waitlist
	request.Waitlist++
	err = s.repo.UpdateRequest(request)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	return request, nil
}

// GetRequest retrieves a request record.
func (s *service) GetRequest(requestID int64) (*data.Request, error) {
	request, err := s.repo.GetRequest(requestID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return request, nil
}

// ListRequests retrieves a paginated list of all requests.
// Records can be filtered and sorted.
func (s *service) ListRequests(search string, status string, filters data.Filters) ([]*data.Request, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	requests, metadata, err := s.repo.GetAllRequests(search, status, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return requests, metadata, nil
}

// SubscribeRequest service subscribes a user to a book request.
func (s *service) SubscribeRequest(userID int64, requestID int64) error {
	// Retrieve the request by its ID
	request, err := s.repo.GetRequest(requestID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	// Subscribe user to the request
	err = s.repo.AddRequestForUser(userID, requestID, time.Now().Add(time.Hour*24*182))
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	// Update request waitlist
	request.Waitlist++
	err = s.repo.UpdateRequest(request)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

// UnsubscribeRequest service unsubscribes a user from a book request.
func (s *service) UnsubscribeRequest(userID int64, requestID int64) error {
	// Retrieve the request by its ID
	request, err := s.repo.GetRequest(requestID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	// Unsubscribe user from the request
	err = s.repo.DeleteRequestForUser(userID, requestID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	// Update request waitlist
	request.Waitlist--
	err = s.repo.UpdateRequest(request)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}
