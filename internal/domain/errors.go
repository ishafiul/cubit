package domain

import "errors"

var (
	// ErrNotFound indicates a requested domain entity does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrConflict indicates an entity with the same identifier or unique attribute already exists.
	ErrConflict = errors.New("resource already exists")

	// ErrValidation indicates an entity invariant or validation rule was violated.
	ErrValidation = errors.New("validation error")

	// ErrInvalidState indicates an illegal state transition was attempted.
	ErrInvalidState = errors.New("invalid entity state transition")
)

// DomainError wraps an error code and a human-readable message.
type DomainError struct {
	Code    string
	Message string
	Err     error
}

func (e *DomainError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewValidationError creates a new validation domain error.
func NewValidationError(message string) error {
	return &DomainError{
		Code:    "VALIDATION_ERROR",
		Message: message,
		Err:     ErrValidation,
	}
}

// NewNotFoundError creates a new not found domain error.
func NewNotFoundError(message string) error {
	return &DomainError{
		Code:    "NOT_FOUND",
		Message: message,
		Err:     ErrNotFound,
	}
}

// NewConflictError creates a new conflict domain error.
func NewConflictError(message string) error {
	return &DomainError{
		Code:    "CONFLICT",
		Message: message,
		Err:     ErrConflict,
	}
}

// NewInvalidStateError creates a new invalid state transition error.
func NewInvalidStateError(message string) error {
	return &DomainError{
		Code:    "INVALID_STATE",
		Message: message,
		Err:     ErrInvalidState,
	}
}
