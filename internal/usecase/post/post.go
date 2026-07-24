package post

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
	imgutil "mini-instagram/pkg/image"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

const (
	MaxCaptionLength = 2048
	DefaultPage      = 1
	DefaultPerPage   = 10
	MaxPerPage       = 100
	MaxHashtags      = 30
	MaxHashtagLength = 64
)

var hashtagPattern = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

// LikeCache is the write-buffer/read-cache the like/unlike usecase talks
// to (T22). A nil likeCache on UseCase means the cache is unavailable
// (e.g. Redis down at startup) and every like/unlike/count/is_liked
// operation falls back to Postgres directly.
type LikeCache interface {
	GetLikeState(ctx context.Context, userID, postID int64) (value string, found bool, err error)
	SetLikeState(ctx context.Context, userID, postID int64, value string) error
	DeleteLikeState(ctx context.Context, userID, postID int64) error
	GetCount(ctx context.Context, postID int64) (count, updatedAt int64, found bool, err error)
	GetCounts(ctx context.Context, postIDs []int64) (counts map[int64]int64, missing []int64, err error)
	InitCount(ctx context.Context, postID, count int64) error
	IncrCount(ctx context.Context, postID int64) error
	DecrCount(ctx context.Context, postID int64) error
	DeleteCount(ctx context.Context, postID int64) error
}

type UseCase struct {
	posts     repo.Post
	hashtags  repo.Hashtag
	likeCache LikeCache
	st        *storage.Storage
	logger    logger.Interface
}

func New(posts repo.Post, hashtags repo.Hashtag, likeCache LikeCache, st *storage.Storage, logger logger.Interface) usecase.Post {
	return &UseCase{posts: posts, hashtags: hashtags, likeCache: likeCache, st: st, logger: logger}
}

// parseHashtags extracts, lowercases, and dedupes hashtags from a caption,
// capping at MaxHashtags and skipping tags longer than MaxHashtagLength.
func parseHashtags(caption string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(caption, -1)

	seen := make(map[string]struct{})
	tags := make([]string, 0, len(matches))
	for _, m := range matches {
		tag := strings.ToLower(m[1])
		if len(tag) > MaxHashtagLength {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
		if len(tags) >= MaxHashtags {
			break
		}
	}
	return tags
}

func (u *UseCase) Create(ctx context.Context, input request.CreatePost) error {
	caption := bluemonday.StrictPolicy().Sanitize(input.Caption)
	if len(caption) > MaxCaptionLength {
		return entity.NewValidationError("caption", fmt.Sprintf("caption exceeds %d characters", MaxCaptionLength))
	}

	if err := imgutil.Validate(input.Header, imgutil.DefaultMaxSize); err != nil {
		return entity.NewValidationError("image", err.Error())
	}

	buf := make([]byte, 512)
	n, err := io.ReadFull(input.File, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("read image content: %w", err)
	}
	buf = buf[:n]

	contentType := http.DetectContentType(buf)
	if !imgutil.IsImage(contentType) {
		return entity.NewValidationError("image", "file must be an image")
	}

	if seeker, ok := input.File.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("rewind image file: %w", err)
		}
	}

	img, err := imgutil.Decode(input.File)
	if err != nil {
		return fmt.Errorf("invalid image content: %w", err)
	}

	name := randomHex(16)
	imagePath := fmt.Sprintf("posts/%s.jpg", name)
	thumbnailPath := fmt.Sprintf("thumbnails/%s.jpg", name)

	originalBuf, err := imgutil.EncodeJPEG(img, 90)
	if err != nil {
		return fmt.Errorf("sanitize image: %w", err)
	}
	imagePath, err = u.st.Save(imagePath, originalBuf)
	if err != nil {
		return fmt.Errorf("save post image: %w", err)
	}

	cleanupImage := true
	defer func() {
		if cleanupImage {
			if err := u.st.Delete(imagePath); err != nil {
				u.logger.Error("failed to cleanup post image", "path", imagePath, "error", err)
			}
		}
	}()

	thumbBuf, err := imgutil.Thumbnail(img, imgutil.ThumbnailMaxSide, imgutil.ThumbnailQuality)
	if err != nil {
		return fmt.Errorf("generate thumbnail: %w", err)
	}
	thumbnailPath, err = u.st.Save(thumbnailPath, thumbBuf)
	if err != nil {
		return fmt.Errorf("save thumbnail: %w", err)
	}

	cleanupThumb := true
	defer func() {
		if cleanupThumb {
			if err := u.st.Delete(thumbnailPath); err != nil {
				u.logger.Error("failed to cleanup thumbnail", "path", thumbnailPath, "error", err)
			}
		}
	}()

	_, err = u.posts.Create(ctx, entity.Post{
		UserID:        input.UserID,
		ImagePath:     imagePath,
		ThumbnailPath: thumbnailPath,
		Caption:       caption,
	}, parseHashtags(caption))
	if err != nil {
		return fmt.Errorf("create post: %w", err)
	}

	cleanupImage = false
	cleanupThumb = false
	return nil
}

func (u *UseCase) GetFeed(ctx context.Context, callerID int64, page, perPage int) (response.Feed, error) {
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

	count, err := u.posts.CountFeed(ctx, callerID)
	if err != nil {
		return response.Feed{}, fmt.Errorf("count feed: %w", err)
	}

	posts, err := u.posts.ListFeed(ctx, callerID, perPage, offset)
	if err != nil {
		return response.Feed{}, fmt.Errorf("list feed: %w", err)
	}

	postIDs := make([]int64, len(posts))
	for i, p := range posts {
		postIDs[i] = p.ID
	}
	likeCounts, err := u.likeCountsCached(ctx, postIDs)
	if err != nil {
		return response.Feed{}, fmt.Errorf("get like counts: %w", err)
	}

	items := make([]response.FeedItem, len(posts))
	for i, p := range posts {
		items[i] = response.FeedItem{
			UserID:        p.UserID,
			Username:      p.Username,
			PostID:        p.ID,
			Caption:       p.Caption,
			ImagePath:     p.ImagePath,
			LikesCount:    likeCounts[p.ID],
			CommentsCount: p.CommentCount,
			CreatedAt:     p.CreatedAt,
		}
	}

	return response.Feed{Count: count, Items: items}, nil
}

func (u *UseCase) Like(ctx context.Context, callerID, postID int64) error {
	if u.likeCache == nil {
		if err := u.posts.Like(ctx, callerID, postID); err != nil {
			return fmt.Errorf("like post: %w", err)
		}
		return nil
	}

	ownerID, err := u.posts.GetOwner(ctx, postID)
	if err != nil {
		return err
	}

	isLiked, err := u.isLikedCached(ctx, callerID, postID)
	if err != nil {
		u.logger.Error("like cache unavailable, falling back to db", "error", err)
		if err := u.posts.Like(ctx, callerID, postID); err != nil {
			return fmt.Errorf("like post: %w", err)
		}
		return nil
	}
	if isLiked {
		return nil
	}

	state, found, err := u.likeCache.GetLikeState(ctx, callerID, postID)
	if err != nil {
		u.logger.Error("like cache unavailable, falling back to db", "error", err)
		if err := u.posts.Like(ctx, callerID, postID); err != nil {
			return fmt.Errorf("like post: %w", err)
		}
		return nil
	}

	if found && state == "0" {
		if err := u.likeCache.DeleteLikeState(ctx, callerID, postID); err != nil {
			u.logger.Error("failed to cancel pending unlike", "post_id", postID, "error", err)
		}
	} else if err := u.likeCache.SetLikeState(ctx, callerID, postID, "1"); err != nil {
		u.logger.Error("failed to set pending like", "post_id", postID, "error", err)
	}

	if err := u.ensureCountInitialized(ctx, postID); err != nil {
		u.logger.Error("failed to initialize like count cache", "post_id", postID, "error", err)
	}
	if err := u.likeCache.IncrCount(ctx, postID); err != nil {
		u.logger.Error("failed to increment like count cache", "post_id", postID, "error", err)
	}

	if ownerID != callerID {
		if err := u.posts.NotifyLike(ctx, ownerID, callerID, postID); err != nil {
			u.logger.Error("failed to create like notification", "post_id", postID, "error", err)
		}
	}

	return nil
}

func (u *UseCase) Unlike(ctx context.Context, callerID, postID int64) error {
	if u.likeCache == nil {
		if err := u.posts.Unlike(ctx, callerID, postID); err != nil {
			return fmt.Errorf("unlike post: %w", err)
		}
		return nil
	}

	if _, err := u.posts.GetOwner(ctx, postID); err != nil {
		return err
	}

	isLiked, err := u.isLikedCached(ctx, callerID, postID)
	if err != nil {
		u.logger.Error("like cache unavailable, falling back to db", "error", err)
		if err := u.posts.Unlike(ctx, callerID, postID); err != nil {
			return fmt.Errorf("unlike post: %w", err)
		}
		return nil
	}
	if !isLiked {
		return entity.ErrNotLiked
	}

	state, found, err := u.likeCache.GetLikeState(ctx, callerID, postID)
	if err != nil {
		u.logger.Error("like cache unavailable, falling back to db", "error", err)
		if err := u.posts.Unlike(ctx, callerID, postID); err != nil {
			return fmt.Errorf("unlike post: %w", err)
		}
		return nil
	}

	if found && state == "1" {
		if err := u.likeCache.DeleteLikeState(ctx, callerID, postID); err != nil {
			u.logger.Error("failed to cancel pending like", "post_id", postID, "error", err)
		}
	} else if err := u.likeCache.SetLikeState(ctx, callerID, postID, "0"); err != nil {
		u.logger.Error("failed to set pending unlike", "post_id", postID, "error", err)
	}

	if err := u.ensureCountInitialized(ctx, postID); err != nil {
		u.logger.Error("failed to initialize like count cache", "post_id", postID, "error", err)
	}
	if err := u.likeCache.DecrCount(ctx, postID); err != nil {
		u.logger.Error("failed to decrement like count cache", "post_id", postID, "error", err)
	}

	return nil
}

// isLikedCached resolves is_liked for (userID, postID) through the like
// cache, falling back to Postgres on a cache miss (normal) or a cache
// error (Redis down).
func (u *UseCase) isLikedCached(ctx context.Context, userID, postID int64) (bool, error) {
	if u.likeCache != nil {
		state, found, err := u.likeCache.GetLikeState(ctx, userID, postID)
		if err == nil {
			if found {
				return state == "1", nil
			}
			return u.posts.IsLiked(ctx, userID, postID)
		}
		u.logger.Error("like cache read failed, falling back to db", "error", err)
	}
	return u.posts.IsLiked(ctx, userID, postID)
}

// likeCountCached resolves the like count for a single post through the
// cache, initializing it from Postgres on a miss.
func (u *UseCase) likeCountCached(ctx context.Context, postID int64) (int64, error) {
	if u.likeCache != nil {
		count, _, found, err := u.likeCache.GetCount(ctx, postID)
		if err == nil {
			if found {
				return count, nil
			}
			counts, err := u.posts.CountLikesBatch(ctx, []int64{postID})
			if err != nil {
				return 0, fmt.Errorf("count likes: %w", err)
			}
			dbCount := counts[postID]
			if err := u.likeCache.InitCount(ctx, postID, dbCount); err != nil {
				u.logger.Error("failed to init like count cache", "post_id", postID, "error", err)
			}
			return dbCount, nil
		}
		u.logger.Error("like cache read failed, falling back to db", "error", err)
	}

	counts, err := u.posts.CountLikesBatch(ctx, []int64{postID})
	if err != nil {
		return 0, fmt.Errorf("count likes: %w", err)
	}
	return counts[postID], nil
}

// likeCountsCached resolves like counts for multiple posts, batching the
// cache reads via a pipeline and the DB fallback via a single grouped
// COUNT query for whatever missed the cache.
func (u *UseCase) likeCountsCached(ctx context.Context, postIDs []int64) (map[int64]int64, error) {
	if len(postIDs) == 0 {
		return map[int64]int64{}, nil
	}

	var counts map[int64]int64
	missing := postIDs

	if u.likeCache != nil {
		cached, miss, err := u.likeCache.GetCounts(ctx, postIDs)
		if err == nil {
			counts = cached
			missing = miss
		} else {
			u.logger.Error("like cache batch read failed, falling back to db", "error", err)
			counts = make(map[int64]int64, len(postIDs))
		}
	} else {
		counts = make(map[int64]int64, len(postIDs))
	}

	if len(missing) == 0 {
		return counts, nil
	}

	dbCounts, err := u.posts.CountLikesBatch(ctx, missing)
	if err != nil {
		return nil, fmt.Errorf("count likes batch: %w", err)
	}
	for _, id := range missing {
		count := dbCounts[id]
		counts[id] = count
		if u.likeCache != nil {
			if err := u.likeCache.InitCount(ctx, id, count); err != nil {
				u.logger.Error("failed to init like count cache", "post_id", id, "error", err)
			}
		}
	}

	return counts, nil
}

// ensureCountInitialized seeds the like count cache from Postgres if it
// hasn't been initialized yet, so IncrCount/DecrCount never operate on an
// empty hash.
func (u *UseCase) ensureCountInitialized(ctx context.Context, postID int64) error {
	_, _, found, err := u.likeCache.GetCount(ctx, postID)
	if err != nil {
		return err
	}
	if found {
		return nil
	}
	counts, err := u.posts.CountLikesBatch(ctx, []int64{postID})
	if err != nil {
		return err
	}
	return u.likeCache.InitCount(ctx, postID, counts[postID])
}

func (u *UseCase) GetByID(ctx context.Context, callerID, postID int64) (response.PostDetail, error) {
	post, err := u.posts.GetByID(ctx, postID)
	if err != nil {
		return response.PostDetail{}, fmt.Errorf("get post: %w", err)
	}

	isLiked, err := u.isLikedCached(ctx, callerID, postID)
	if err != nil {
		return response.PostDetail{}, fmt.Errorf("check is liked: %w", err)
	}

	likesCount, err := u.likeCountCached(ctx, postID)
	if err != nil {
		return response.PostDetail{}, fmt.Errorf("get like count: %w", err)
	}

	return response.PostDetail{
		PostID:        post.ID,
		UserID:        post.UserID,
		Username:      post.Username,
		Caption:       post.Caption,
		ImagePath:     post.ImagePath,
		LikesCount:    likesCount,
		CommentsCount: post.CommentCount,
		CreatedAt:     post.CreatedAt,
		IsLiked:       isLiked,
	}, nil
}

func (u *UseCase) Delete(ctx context.Context, callerID, postID int64) error {
	post, err := u.posts.GetForDelete(ctx, postID)
	if err != nil {
		return fmt.Errorf("get post for delete: %w", err)
	}
	if post.UserID != callerID {
		return entity.ErrForbidden
	}

	if err := u.posts.SoftDelete(ctx, postID); err != nil {
		return fmt.Errorf("soft delete post: %w", err)
	}

	if u.likeCache != nil {
		if err := u.likeCache.DeleteCount(ctx, postID); err != nil {
			u.logger.Error("failed to delete like count cache", "post_id", postID, "error", err)
		}
	}

	if err := u.st.Delete(post.ImagePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		u.logger.Error("failed to delete post image", "path", post.ImagePath, "error", err)
	}
	if post.ThumbnailPath != "" {
		if err := u.st.Delete(post.ThumbnailPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			u.logger.Error("failed to delete post thumbnail", "path", post.ThumbnailPath, "error", err)
		}
	}
	return nil
}

func (u *UseCase) Edit(ctx context.Context, callerID, postID int64, caption string) error {
	post, err := u.posts.GetForDelete(ctx, postID)
	if err != nil {
		return fmt.Errorf("get post for edit: %w", err)
	}
	if post.UserID != callerID {
		return entity.ErrForbidden
	}

	caption = bluemonday.StrictPolicy().Sanitize(caption)
	if len(caption) > MaxCaptionLength {
		return entity.NewValidationError("caption", fmt.Sprintf("caption exceeds %d characters", MaxCaptionLength))
	}

	if err := u.posts.UpdateCaption(ctx, postID, caption, parseHashtags(caption)); err != nil {
		return fmt.Errorf("update post caption: %w", err)
	}
	return nil
}

func (u *UseCase) SearchByTag(ctx context.Context, tag string, page, perPage int) (response.HashtagPostList, error) {
	tag = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(tag), "#"))
	if tag == "" {
		return response.HashtagPostList{}, entity.NewValidationError("tag", "tag is required")
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

	count, err := u.hashtags.CountByTag(ctx, tag)
	if err != nil {
		return response.HashtagPostList{}, fmt.Errorf("count posts by hashtag: %w", err)
	}

	posts, err := u.hashtags.ListByTag(ctx, tag, perPage, offset)
	if err != nil {
		return response.HashtagPostList{}, fmt.Errorf("list posts by hashtag: %w", err)
	}

	postIDs := make([]int64, len(posts))
	for i, p := range posts {
		postIDs[i] = p.ID
	}
	likeCounts, err := u.likeCountsCached(ctx, postIDs)
	if err != nil {
		return response.HashtagPostList{}, fmt.Errorf("get like counts: %w", err)
	}

	items := make([]response.HashtagPostItem, len(posts))
	for i, p := range posts {
		items[i] = response.HashtagPostItem{
			PostID:        p.ID,
			UserID:        p.UserID,
			Username:      p.Username,
			ThumbnailPath: p.ThumbnailPath,
			Caption:       p.Caption,
			LikesCount:    likeCounts[p.ID],
			CommentsCount: p.CommentCount,
			CreatedAt:     p.CreatedAt,
		}
	}

	return response.HashtagPostList{Count: count, Items: items}, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random hex: %v", err))
	}
	return hex.EncodeToString(b)
}
