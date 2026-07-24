package user

import (
	"context"
	"fmt"
	"strings"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
	"mini-instagram/internal/validation"
	"mini-instagram/pkg/image"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"

	"github.com/microcosm-cc/bluemonday"
)

const (
	DefaultPage          = 1
	DefaultPerPage       = 10
	MaxPerPage           = 100
	MaxAvatarSize        = 5 << 20 // 5 MB
	MaxSearchQueryLength = 32
)

type UseCase struct {
	users  repo.User
	posts  repo.Post
	st     *storage.Storage
	logger logger.Interface
}

func New(users repo.User, posts repo.Post, st *storage.Storage, l logger.Interface) usecase.User {
	return &UseCase{users: users, posts: posts, st: st, logger: l}
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
			LikesCount:    p.LikeCount,
			CommentsCount: p.CommentCount,
			CreatedAt:     p.CreatedAt,
		}
	}

	return response.UserPosts{Count: count, Items: items}, nil
}

func (u *UseCase) UpdateProfile(ctx context.Context, input request.UpdateProfile) error {
	sanitizer := bluemonday.StrictPolicy()

	username := strings.TrimSpace(strings.ToLower(input.Username))
	fullName := strings.TrimSpace(sanitizer.Sanitize(input.FullName))
	bio := strings.TrimSpace(sanitizer.Sanitize(input.Bio))

	if err := validation.Profile(username, fullName, bio); err != nil {
		return err
	}

	user, err := u.users.FindByID(ctx, input.UserID)
	if err != nil {
		return err
	}

	if username != user.Username {
		taken, err := u.users.UsernameExists(ctx, username)
		if err != nil {
			return fmt.Errorf("check username: %w", err)
		}
		if taken {
			return entity.ErrUsernameTaken
		}
	}

	var newAvatarPath string
	oldAvatarPath := user.AvatarPath

	if input.Avatar != nil {
		if err := image.Validate(input.AvatarHeader, MaxAvatarSize); err != nil {
			return entity.NewValidationError("avatar", err.Error())
		}
		savedPath, err := image.Save(input.Avatar, input.AvatarHeader, u.st, "avatars", MaxAvatarSize)
		if err != nil {
			return fmt.Errorf("save avatar: %w", err)
		}
		newAvatarPath = savedPath
	} else {
		newAvatarPath = oldAvatarPath
	}

	user.Username = username
	user.FullName = fullName
	user.Bio = bio
	user.AvatarPath = newAvatarPath

	if err := u.users.Update(ctx, user); err != nil {
		if input.Avatar != nil && newAvatarPath != "" {
			if delErr := u.st.Delete(newAvatarPath); delErr != nil {
				u.logger.Error("failed to cleanup new avatar after update error", "path", newAvatarPath, "error", delErr)
			}
		}
		return err
	}

	if input.Avatar != nil && oldAvatarPath != "" && oldAvatarPath != newAvatarPath {
		if err := u.st.Delete(oldAvatarPath); err != nil {
			u.logger.Error("failed to delete old avatar", "path", oldAvatarPath, "error", err)
		}
	}

	return nil
}

func (u *UseCase) Follow(ctx context.Context, followerID, followingID int64) error {
	if followerID == followingID {
		return entity.ErrSelfFollow
	}

	target, err := u.users.FindByID(ctx, followingID)
	if err != nil {
		return err
	}
	if !target.IsActive {
		return entity.ErrNotFound
	}

	return u.users.Follow(ctx, followerID, followingID)
}

func (u *UseCase) Unfollow(ctx context.Context, followerID, followingID int64) error {
	target, err := u.users.FindByID(ctx, followingID)
	if err != nil {
		return err
	}
	if !target.IsActive {
		return entity.ErrNotFound
	}

	return u.users.Unfollow(ctx, followerID, followingID)
}

func (u *UseCase) SearchUsers(ctx context.Context, callerID int64, query string, page, perPage int) (response.UserSearch, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return response.UserSearch{}, entity.NewValidationError("q", "q is required")
	}
	if len(q) > MaxSearchQueryLength {
		return response.UserSearch{}, entity.NewValidationError("q", fmt.Sprintf("q must be at most %d characters", MaxSearchQueryLength))
	}

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

	likePattern := escapeLike(q)

	count, err := u.users.CountSearch(ctx, likePattern)
	if err != nil {
		return response.UserSearch{}, fmt.Errorf("count search users: %w", err)
	}

	users, err := u.users.Search(ctx, likePattern, strings.ToLower(q), perPage, offset)
	if err != nil {
		return response.UserSearch{}, fmt.Errorf("search users: %w", err)
	}

	items := make([]response.UserSearchItem, len(users))
	for i, usr := range users {
		isFollowing := false
		if callerID != usr.ID {
			isFollowing, err = u.users.IsFollowing(ctx, callerID, usr.ID)
			if err != nil {
				return response.UserSearch{}, fmt.Errorf("check is following: %w", err)
			}
		}

		items[i] = response.UserSearchItem{
			UserID:      usr.ID,
			Username:    usr.Username,
			FullName:    usr.FullName,
			AvatarPath:  usr.AvatarPath,
			IsFollowing: isFollowing,
		}
	}

	return response.UserSearch{Count: count, Items: items}, nil
}

// escapeLike escapes LIKE/ILIKE pattern metacharacters so user input is
// treated as a literal substring rather than a wildcard pattern.
func escapeLike(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(s)
}
