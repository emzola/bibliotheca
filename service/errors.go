package service

import (
	"errors"
	"fmt"
)

var (
	ErrFailedValidation     = errors.New("failed validation")
	ErrRecordNotFound       = errors.New("record not found")
	ErrEditConflict         = errors.New("edit conflict")
	ErrPasswordMismatch     = errors.New("password mismatch")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnsupportedMediaType = errors.New("unsupported media type")
	ErrContentTooLarge      = errors.New("content too large")
	ErrBadRequest           = errors.New("bad request")
	ErrDuplicateRecord      = errors.New("duplicate record")
	ErrNotPermitted         = errors.New("not permitted")
)

// failedValidation loops through a validation error map and
// returns an error string with the key and value of the map.
func (s *service) failedValidation(errorMap map[string]string) error {
	var err error
	for k, v := range errorMap {
		err = fmt.Errorf("%q %s", k, v)
	}
	return err
}
