package post

import (
	"context"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/storage"
)

type fakePostRepo struct {
	created entity.Post
	err     error
}

func (f *fakePostRepo) Create(ctx context.Context, post entity.Post) (entity.Post, error) {
	if f.err != nil {
		return entity.Post{}, f.err
	}
	f.created = post
	f.created.ID = 1
	return f.created, nil
}

func newTestStorage(t *testing.T) *storage.Storage {
	t.Helper()
	dir := t.TempDir()
	return storage.New(dir)
}

func mustOpenTestImage(t *testing.T) (*os.File, *multipart.FileHeader) {
	t.Helper()

	f, err := os.CreateTemp("", "test-image-*.jpg")
	if err != nil {
		t.Fatalf("create temp image: %v", err)
	}
	defer os.Remove(f.Name())

	img := image.NewRGBA(image.Rect(0, 0, 500, 300))
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatalf("encode test image: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek temp image: %v", err)
	}

	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat temp image: %v", err)
	}

	header := &multipart.FileHeader{
		Filename: filepath.Base(f.Name()),
		Header:   map[string][]string{"Content-Type": {"image/jpeg"}},
		Size:     fi.Size(),
	}
	return f, header
}

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func TestCreatePost_Success(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "hello", File: file, Header: header})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.created.ID != 1 {
		t.Fatalf("expected post id 1, got %d", repo.created.ID)
	}
	if repo.created.UserID != 1 {
		t.Fatalf("expected user id 1, got %d", repo.created.UserID)
	}
	if repo.created.Caption != "hello" {
		t.Fatalf("expected caption hello, got %s", repo.created.Caption)
	}
	if repo.created.ImagePath == "" {
		t.Fatal("expected non-empty image path")
	}
	if repo.created.ThumbnailPath == "" {
		t.Fatal("expected non-empty thumbnail path")
	}

	if _, err := os.Stat(st.FullPath(repo.created.ImagePath)); err != nil {
		t.Fatalf("image file missing: %v", err)
	}
	if _, err := os.Stat(st.FullPath(repo.created.ThumbnailPath)); err != nil {
		t.Fatalf("thumbnail file missing: %v", err)
	}
}

type fakeFile struct {
	io.ReadSeeker
}

func (fakeFile) ReadAt(p []byte, off int64) (int, error) { return 0, io.EOF }
func (fakeFile) Close() error                            { return nil }

func TestCreatePost_InvalidImage(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, st, nopLogger{})

	file := fakeFile{ReadSeeker: strings.NewReader("not an image")}
	header := &multipart.FileHeader{
		Filename: "test.txt",
		Header:   map[string][]string{"Content-Type": {"text/plain"}},
		Size:     12,
	}

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "", File: file, Header: header})
	if err == nil {
		t.Fatal("expected error for invalid image")
	}
}

func TestCreatePost_CaptionSanitizesHTML(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "<script>alert(1)</script>hello", File: file, Header: header})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if strings.Contains(repo.created.Caption, "<script>") {
		t.Fatalf("expected HTML tags to be stripped, got %q", repo.created.Caption)
	}
	if repo.created.Caption != "hello" {
		t.Fatalf("expected caption 'hello', got %q", repo.created.Caption)
	}
}

func TestCreatePost_CaptionTooLong(t *testing.T) {
	repo := &fakePostRepo{}
	st := newTestStorage(t)
	uc := New(repo, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: strings.Repeat("a", MaxCaptionLength+1), File: file, Header: header})
	if err == nil {
		t.Fatal("expected error for long caption")
	}
}

func TestCreatePost_CleanupOnRepoFailure(t *testing.T) {
	repo := &fakePostRepo{err: errors.New("db failure")}
	st := newTestStorage(t)
	uc := New(repo, st, nopLogger{})

	file, header := mustOpenTestImage(t)
	defer file.Close()

	err := uc.Create(context.Background(), request.CreatePost{UserID: 1, Caption: "hello", File: file, Header: header})
	if err == nil {
		t.Fatal("expected error")
	}

	entries, err := os.ReadDir(st.FullPath("posts"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read posts dir: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected cleanup of image file, found %d entries", len(entries))
	}
}
