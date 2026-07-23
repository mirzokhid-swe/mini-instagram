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
)

type UseCase struct {
	posts  repo.Post
	st     *storage.Storage
	logger logger.Interface
}

func New(posts repo.Post, st *storage.Storage, logger logger.Interface) usecase.Post {
	return &UseCase{posts: posts, st: st, logger: logger}
}

func (u *UseCase) Create(ctx context.Context, input request.CreatePost) error {
	caption := bluemonday.StrictPolicy().Sanitize(input.Caption)
	if len(caption) > MaxCaptionLength {
		return fmt.Errorf("caption exceeds %d characters", MaxCaptionLength)
	}

	if err := imgutil.Validate(input.Header, imgutil.DefaultMaxSize); err != nil {
		return err
	}

	buf := make([]byte, 512)
	n, err := io.ReadFull(input.File, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("read image content: %w", err)
	}
	buf = buf[:n]

	contentType := http.DetectContentType(buf)
	if !imgutil.IsImage(contentType) {
		return errors.New("file must be an image")
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
	})
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

	items := make([]response.FeedItem, len(posts))
	for i, p := range posts {
		items[i] = response.FeedItem{
			UserID:        p.UserID,
			Username:      p.Username,
			PostID:        p.ID,
			Caption:       p.Caption,
			ImagePath:     p.ImagePath,
			LikesCount:    p.LikeCount,
			CommentsCount: p.CommentCount,
			CreatedAt:     p.CreatedAt,
		}
	}

	return response.Feed{Count: count, Items: items}, nil
}

func (u *UseCase) Like(ctx context.Context, callerID, postID int64) error {
	if err := u.posts.Like(ctx, callerID, postID); err != nil {
		return fmt.Errorf("like post: %w", err)
	}
	return nil
}

func (u *UseCase) Unlike(ctx context.Context, callerID, postID int64) error {
	if err := u.posts.Unlike(ctx, callerID, postID); err != nil {
		return fmt.Errorf("unlike post: %w", err)
	}
	return nil
}

func (u *UseCase) GetByID(ctx context.Context, callerID, postID int64) (response.PostDetail, error) {
	post, err := u.posts.GetByID(ctx, postID)
	if err != nil {
		return response.PostDetail{}, fmt.Errorf("get post: %w", err)
	}

	isLiked, err := u.posts.IsLiked(ctx, callerID, postID)
	if err != nil {
		return response.PostDetail{}, fmt.Errorf("check is liked: %w", err)
	}

	return response.PostDetail{
		PostID:        post.ID,
		UserID:        post.UserID,
		Username:      post.Username,
		Caption:       post.Caption,
		ImagePath:     post.ImagePath,
		LikesCount:    post.LikeCount,
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

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random hex: %v", err))
	}
	return hex.EncodeToString(b)
}
