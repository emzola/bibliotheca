package repository

import "errors"

var (
	ErrRecordNotFound   = errors.New("record not found")
	ErrFailedValidation = errors.New("failed validation")
	ErrEditConflict     = errors.New("edit conflict")
	ErrDuplicateRecord  = errors.New("duplicate record")
	ErrNotPermitted     = errors.New("not permitted")
	ErrBadRequest       = errors.New("bad request")
)
