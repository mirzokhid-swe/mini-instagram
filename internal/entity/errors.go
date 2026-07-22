package entity

import "errors"

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already exists")
	ErrUsernameTaken      = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)
