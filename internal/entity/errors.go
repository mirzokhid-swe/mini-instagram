package entity

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already exists")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrPostNotFound       = errors.New("post not found")
	ErrNotLiked           = errors.New("post is not liked")
)

// ValidationError marks an error as caused by invalid caller input, as
// opposed to an internal failure. Controllers use errors.As to detect it
// and return the message to the client instead of a generic 500.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
