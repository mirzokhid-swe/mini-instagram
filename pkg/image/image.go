// Package image provides helpers for image upload validation and storage.
package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"

	"mini-instagram/pkg/storage"
)

// DefaultMaxSize is the default maximum allowed image file size (10 MB).
const DefaultMaxSize = 10 << 20

// Validate checks that the uploaded file is an image and does not exceed maxSize.
func Validate(header *multipart.FileHeader, maxSize int64) error {
	if header.Size > maxSize {
		return fmt.Errorf("image must be at most %d MB", maxSize>>20)
	}

	contentType := header.Header.Get("Content-Type")
	if !IsImage(contentType) {
		return fmt.Errorf("file must be an image")
	}

	return nil
}

// IsImage reports whether the given content type is a supported image format.
func IsImage(contentType string) bool {
	return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
}

// SanitizeFilename removes path traversal characters from a filename.
func SanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, "..", "")
	name = strings.ReplaceAll(name, "/", "")
	name = strings.ReplaceAll(name, "\\", "")
	if name == "" {
		return "file"
	}
	return name
}

// Save reads the uploaded image from r, validates it, and writes it to storage under subdir.
// It returns the relative path of the saved file.
func Save(file multipart.File, header *multipart.FileHeader, st *storage.Storage, subdir string, maxSize int64) (string, error) {
	if err := Validate(header, maxSize); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s/%d_%s", subdir, time.Now().UnixNano(), SanitizeFilename(filepath.Base(header.Filename)))
	imagePath, err := st.Save(filename, file)
	if err != nil {
		return "", fmt.Errorf("save image: %w", err)
	}

	return imagePath, nil
}

// ReadSeeker is a helper interface used when the caller needs to rewind a multipart file.
// multipart.File already implements this.
type ReadSeeker interface {
	io.Reader
	io.Seeker
}

const ThumbnailMaxSide = 320
const ThumbnailQuality = 80

// Decode reads and decodes an image from r using registered formats (jpeg, png, webp).
func Decode(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// EncodeJPEG encodes img as a JPEG with the given quality and returns it as a buffer.
func EncodeJPEG(img image.Image, quality int) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf, nil
}

// Thumbnail resizes img so that its largest side is at most maxSide pixels,
// preserving aspect ratio, and encodes the result as a JPEG with the given quality.
func Thumbnail(img image.Image, maxSide, quality int) (*bytes.Buffer, error) {
	bounds := img.Bounds()
	if bounds.Dx() > maxSide || bounds.Dy() > maxSide {
		img = imaging.Fit(img, maxSide, maxSide, imaging.Lanczos)
	}
	return EncodeJPEG(img, quality)
}

// GenerateThumbnail opens the image at srcPath, resizes it so that its largest
// side is at most maxSide pixels while preserving the aspect ratio, and writes
// it as a JPEG to dstPath with the given quality.
func GenerateThumbnail(srcPath, dstPath string, maxSide, quality int) error {
	src, err := imaging.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open image for thumbnail: %w", err)
	}

	buf, err := Thumbnail(src, maxSide, quality)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("create thumbnail directory: %w", err)
	}

	file, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create thumbnail file: %w", err)
	}
	defer file.Close()

	if _, err := buf.WriteTo(file); err != nil {
		return fmt.Errorf("write thumbnail file: %w", err)
	}

	return nil
}
