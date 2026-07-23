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
	GetFeed(ctx context.Context, callerID int64, page, perPage int) (response.Feed, error)
	Like(ctx context.Context, callerID, postID int64) error
	Unlike(ctx context.Context, callerID, postID int64) error
	GetByID(ctx context.Context, callerID, postID int64) (response.PostDetail, error)
	Delete(ctx context.Context, callerID, postID int64) error
}

type Comment interface {
	Create(ctx context.Context, callerID, postID int64, content string) error
	List(ctx context.Context, postID int64, page, perPage int) (response.CommentList, error)
	Delete(ctx context.Context, callerID, commentID int64) error
}

type User interface {
	GetProfile(ctx context.Context, userID, callerID int64) (response.Profile, error)
	GetUserPosts(ctx context.Context, userID int64, page, perPage int) (response.UserPosts, error)
	UpdateProfile(ctx context.Context, input request.UpdateProfile) error
}
