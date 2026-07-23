package v1

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

type fakePostUseCase struct {
	called bool
	err    error
}

func (f *fakePostUseCase) Create(ctx context.Context, input request.CreatePost) error {
	f.called = true
	return f.err
}

func (f *fakePostUseCase) GetFeed(ctx context.Context, callerID int64, page, perPage int) (response.Feed, error) {
	return response.Feed{}, nil
}

func newTestPostHandler(uc *fakePostUseCase) (*gin.Engine, *storage.Storage) {
	gin.SetMode(gin.TestMode)
	handler := gin.New()
	st := storage.New("test-media")
	h := &V1{posts: uc, logger: logger.New("error"), storage: st}

	// simulate authenticated user
	auth := handler.Group("/api/v1")
	auth.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
	})
	{
		auth.POST("/post", h.createPost)
	}
	return handler, st
}

func createMultipartImage(t *testing.T) (*os.File, int64) {
	t.Helper()
	f, err := os.CreateTemp("", "test-*.jpg")
	if err != nil {
		t.Fatalf("create temp image: %v", err)
	}
	defer os.Remove(f.Name())

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	if err := jpeg.Encode(f, img, nil); err != nil {
		t.Fatalf("encode image: %v", err)
	}
	fi, err := f.Stat()
	if err != nil {
		t.Fatalf("stat image: %v", err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatalf("seek image: %v", err)
	}
	return f, fi.Size()
}

func TestCreatePostHandler_Success(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	img, _ := createMultipartImage(t)
	defer img.Close()

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, _ := mw.CreateFormFile("image", filepath.Base(img.Name()))
	imgContent, _ := os.Open(img.Name())
	io.Copy(part, imgContent)
	imgContent.Close()
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if !uc.called {
		t.Fatal("expected usecase to be called")
	}
}

func TestCreatePostHandler_Unauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	uc := &fakePostUseCase{}
	h := &V1{posts: uc, logger: logger.New("error")}
	router.POST("/api/v1/post", h.createPost)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
