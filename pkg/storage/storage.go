// Package storage provides local filesystem storage for uploaded media.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrFileNotExists is retained for callers that need to report missing files.
var ErrFileNotExists = fmt.Errorf("file does not exist")

type Storage struct {
	basePath string
}

func New(basePath string) *Storage {
	return &Storage{basePath: basePath}
}

// Save writes file content to filename under basePath.
// It creates the destination file and any missing parent directories.
func (s *Storage) Save(filename string, content io.Reader) (string, error) {
	destPath := filepath.Join(s.basePath, filename)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	file, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, content); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return filename, nil
}
