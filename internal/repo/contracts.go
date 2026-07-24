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
	CountSearch(ctx context.Context, likePattern string) (int64, error)
	Search(ctx context.Context, likePattern, exactMatch string, limit, offset int) ([]entity.User, error)
	CountFollowers(ctx context.Context, userID int64) (int64, error)
	ListFollowers(ctx context.Context, userID int64, limit, offset int) ([]entity.User, error)
	CountFollowing(ctx context.Context, userID int64) (int64, error)
	ListFollowing(ctx context.Context, userID int64, limit, offset int) ([]entity.User, error)
}

type Post interface {
	Create(ctx context.Context, post entity.Post, hashtags []string) (entity.Post, error)
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
	UpdateCaption(ctx context.Context, postID int64, caption string, hashtags []string) error

	// GetOwner returns the post's owner id, or entity.ErrPostNotFound if the
	// post is missing or soft-deleted. Used by the cache-backed like/unlike
	// path (T22) to run the existence check before touching Redis.
	GetOwner(ctx context.Context, postID int64) (ownerID int64, err error)
	// NotifyLike creates a like notification outside of any DB transaction;
	// used by the cache-backed like path, which writes the like itself to
	// Redis instead of the likes table.
	NotifyLike(ctx context.Context, ownerID, actorID, postID int64) error
	// CountLikesBatch returns like counts from the likes table (the source
	// of truth once the like cache is in front of it) for the given post
	// ids. Posts with zero likes are omitted from the result map.
	CountLikesBatch(ctx context.Context, postIDs []int64) (map[int64]int64, error)

	// InsertLikeRow, DeleteLikeRow and UpdateLikeCount are raw, untransacted
	// operations used only by the like-cache flush worker (T22) to replay
	// buffered Redis events into Postgres.
	InsertLikeRow(ctx context.Context, userID, postID int64) error
	DeleteLikeRow(ctx context.Context, userID, postID int64) error
	UpdateLikeCount(ctx context.Context, postID, count int64) error
}

type Comment interface {
	Create(ctx context.Context, comment entity.Comment) error
	Count(ctx context.Context, postID int64) (int64, error)
	List(ctx context.Context, postID int64, limit, offset int) ([]entity.Comment, error)
	GetForDelete(ctx context.Context, commentID int64) (entity.CommentOwnership, error)
	SoftDelete(ctx context.Context, commentID, postID int64) error
	UpdateContent(ctx context.Context, commentID int64, content string) error
}

type Hashtag interface {
	CountByTag(ctx context.Context, tag string) (int64, error)
	ListByTag(ctx context.Context, tag string, limit, offset int) ([]entity.HashtagPost, error)
}

type Notification interface {
	Count(ctx context.Context, userID int64) (int64, error)
	List(ctx context.Context, userID int64, limit, offset int) ([]entity.Notification, error)
	MarkRead(ctx context.Context, notificationID, userID int64) error
}
