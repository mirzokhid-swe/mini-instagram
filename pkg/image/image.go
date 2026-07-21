// Package image provides helpers for image upload validation and storage.
package image

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

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
