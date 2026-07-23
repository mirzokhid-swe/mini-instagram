// Package usecase implements application business logic. Each logic group in own file.
package usecase

import (
	"context"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/controller/restapi/v1/response"
)

type Auth interface {
	CheckSignUpAvailability(ctx context.Context, email, username string) error
	SignUp(ctx context.Context, input request.SignUp) (string, error)
	Login(ctx context.Context, input request.Login) (string, error)
}

type Post interface {
	Create(ctx context.Context, input request.CreatePost) error
}

type User interface {
	GetProfile(ctx context.Context, userID, callerID int64) (response.Profile, error)
	GetUserPosts(ctx context.Context, userID int64, page, perPage int) (response.UserPosts, error)
}
