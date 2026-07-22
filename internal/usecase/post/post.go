package post

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/microcosm-cc/bluemonday"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
	imgutil "mini-instagram/pkg/image"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

const MaxCaptionLength = 2048

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

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random hex: %v", err))
	}
	return hex.EncodeToString(b)
}
