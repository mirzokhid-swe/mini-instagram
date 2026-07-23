package repo

import (
	"context"

	"mini-instagram/internal/entity"
)

type User interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	FindByEmail(ctx context.Context, email string) (entity.User, error)
	FindByID(ctx context.Context, id int64) (entity.User, error)
	Create(ctx context.Context, user entity.User) (entity.User, error)
	Update(ctx context.Context, user entity.User) error
	GetProfileStats(ctx context.Context, userID int64) (postsCount, followersCount, followingCount int64, err error)
	IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error)
	Follow(ctx context.Context, followerID, followingID int64) error
	Unfollow(ctx context.Context, followerID, followingID int64) error
}

type Post interface {
	Create(ctx context.Context, post entity.Post) (entity.Post, error)
	CountByUser(ctx context.Context, userID int64) (int64, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]entity.Post, error)
	CountFeed(ctx context.Context, callerID int64) (int64, error)
	ListFeed(ctx context.Context, callerID int64, limit, offset int) ([]entity.FeedPost, error)
	Like(ctx context.Context, userID, postID int64) error
	Unlike(ctx context.Context, userID, postID int64) error
	GetByID(ctx context.Context, postID int64) (entity.PostDetail, error)
	IsLiked(ctx context.Context, userID, postID int64) (bool, error)
	GetForDelete(ctx context.Context, postID int64) (entity.Post, error)
	SoftDelete(ctx context.Context, postID int64) error
}

type Comment interface {
	Create(ctx context.Context, comment entity.Comment) error
	Count(ctx context.Context, postID int64) (int64, error)
	List(ctx context.Context, postID int64, limit, offset int) ([]entity.Comment, error)
	GetForDelete(ctx context.Context, commentID int64) (entity.CommentOwnership, error)
	SoftDelete(ctx context.Context, commentID, postID int64) error
}
