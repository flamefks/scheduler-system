package apperrors

import "errors"

var (
	ErrNotFound       = errors.New("not found")
	ErrInvalidStatus  = errors.New("invalid status")
	ErrStatusConflict = errors.New("status conflict")
	ErrInvalidJSON    = errors.New("invalid json")
	ErrInvalidUUID    = errors.New("invalid uuid")
)
