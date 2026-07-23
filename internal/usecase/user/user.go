package user

import (
	"context"
	"fmt"

	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 10
	MaxPerPage     = 100
)

type UseCase struct {
	users repo.User
	posts repo.Post
}

func New(users repo.User, posts repo.Post) usecase.User {
	return &UseCase{users: users, posts: posts}
}

func (u *UseCase) GetProfile(ctx context.Context, userID, callerID int64) (response.Profile, error) {
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return response.Profile{}, err
	}
	if !user.IsActive {
		return response.Profile{}, entity.ErrNotFound
	}

	postsCount, followersCount, followingCount, err := u.users.GetProfileStats(ctx, userID)
	if err != nil {
		return response.Profile{}, err
	}

	isFollowing := false
	if callerID != userID {
		isFollowing, err = u.users.IsFollowing(ctx, callerID, userID)
		if err != nil {
			return response.Profile{}, err
		}
	}

	return response.Profile{
		UserID:         user.ID,
		Username:       user.Username,
		FullName:       user.FullName,
		Bio:            user.Bio,
		AvatarPath:     user.AvatarPath,
		PostsCount:     postsCount,
		FollowersCount: followersCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
	}, nil
}

func (u *UseCase) GetUserPosts(ctx context.Context, userID int64, page, perPage int) (response.UserPosts, error) {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	offset := (page - 1) * perPage

	count, err := u.posts.CountByUser(ctx, userID)
	if err != nil {
		return response.UserPosts{}, fmt.Errorf("count user posts: %w", err)
	}

	posts, err := u.posts.ListByUser(ctx, userID, perPage, offset)
	if err != nil {
		return response.UserPosts{}, fmt.Errorf("list user posts: %w", err)
	}

	items := make([]response.PostItem, len(posts))
	for i, p := range posts {
		items[i] = response.PostItem{
			PostID:        p.ID,
			ThumbnailPath: p.ThumbnailPath,
			Caption:       p.Caption,
			CreatedAt:     p.CreatedAt,
		}
	}

	return response.UserPosts{Count: count, Items: items}, nil
}
