// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"mini-instagram/internal/controller/restapi/v1/request"
)

type Auth interface {
	CheckSignUpAvailability(ctx context.Context, email, username string) error
	SignUp(ctx context.Context, input request.SignUp) (string, error)
}
