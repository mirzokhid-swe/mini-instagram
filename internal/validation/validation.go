// Package validation holds the field-level input rules shared by the auth
// and user usecases, so signup and profile editing can't drift apart.
package validation

import (
	"fmt"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
)

const (
	MinUsername  = 3
	MaxUsername  = 32
	MaxFullName  = 64
	MaxBioLength = 512
	MinPassword  = 8
)

func Email(v string) error {
	if v == "" {
		return entity.NewValidationError("email", "email is required")
	}
	return nil
}

func Username(v string) error {
	if len(v) < MinUsername || len(v) > MaxUsername {
		return entity.NewValidationError("username", fmt.Sprintf("username must be between %d and %d characters", MinUsername, MaxUsername))
	}
	return nil
}

func FullName(v string) error {
	if v == "" {
		return entity.NewValidationError("full_name", "full_name is required")
	}
	if len(v) > MaxFullName {
		return entity.NewValidationError("full_name", fmt.Sprintf("full_name must be at most %d characters", MaxFullName))
	}
	return nil
}

func Bio(v string) error {
	if len(v) > MaxBioLength {
		return entity.NewValidationError("bio", fmt.Sprintf("bio must be at most %d characters", MaxBioLength))
	}
	return nil
}

func Password(v string) error {
	if v == "" {
		return entity.NewValidationError("password", "password is required")
	}
	if len(v) < MinPassword {
		return entity.NewValidationError("password", fmt.Sprintf("password must be at least %d characters", MinPassword))
	}
	return nil
}

// SignUp validates a signup request. Fields are checked in the order a
// user filled them in, so the first error reported is the first one they'd
// need to fix.
func SignUp(req request.SignUp) error {
	if err := Email(req.Email); err != nil {
		return err
	}
	if err := FullName(req.FullName); err != nil {
		return err
	}
	if err := Username(req.Username); err != nil {
		return err
	}
	if err := Password(req.Password); err != nil {
		return err
	}
	return Bio(req.Bio)
}

func Login(req request.Login) error {
	if err := Email(req.Email); err != nil {
		return err
	}
	if req.Password == "" {
		return entity.NewValidationError("password", "password is required")
	}
	return nil
}

// Profile validates the editable fields of a profile update.
func Profile(username, fullName, bio string) error {
	if err := Username(username); err != nil {
		return err
	}
	if err := FullName(fullName); err != nil {
		return err
	}
	return Bio(bio)
}
