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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

type fakePostUseCase struct {
	called bool
	err    error

	likeErr, unlikeErr             error
	lastLikePostID, lastUnlikePost int64

	getByIDResp   response.PostDetail
	getByIDErr    error
	deleteErr     error
	lastDeletePID int64

	editErr         error
	lastEditPID     int64
	lastEditCaption string
}

func (f *fakePostUseCase) Create(ctx context.Context, input request.CreatePost) error {
	f.called = true
	return f.err
}

func (f *fakePostUseCase) GetFeed(ctx context.Context, callerID int64, page, perPage int) (response.Feed, error) {
	return response.Feed{}, nil
}

func (f *fakePostUseCase) Like(ctx context.Context, callerID, postID int64) error {
	f.lastLikePostID = postID
	return f.likeErr
}

func (f *fakePostUseCase) Unlike(ctx context.Context, callerID, postID int64) error {
	f.lastUnlikePost = postID
	return f.unlikeErr
}

func (f *fakePostUseCase) GetByID(ctx context.Context, callerID, postID int64) (response.PostDetail, error) {
	return f.getByIDResp, f.getByIDErr
}

func (f *fakePostUseCase) Delete(ctx context.Context, callerID, postID int64) error {
	f.lastDeletePID = postID
	return f.deleteErr
}

func (f *fakePostUseCase) Edit(ctx context.Context, callerID, postID int64, caption string) error {
	f.lastEditPID = postID
	f.lastEditCaption = caption
	return f.editErr
}

func (f *fakePostUseCase) SearchByTag(ctx context.Context, tag string, page, perPage int) (response.HashtagPostList, error) {
	return response.HashtagPostList{}, nil
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
		auth.GET("/post/:post_id", h.getPost)
		auth.PUT("/post/:post_id", h.editPost)
		auth.DELETE("/post/:post_id", h.deletePost)
		auth.POST("/post/:post_id/like", h.likePost)
		auth.DELETE("/post/:post_id/like", h.unlikePost)
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

func TestLikePostHandler_Success(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/42/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastLikePostID != 42 {
		t.Fatalf("expected post_id 42, got %d", uc.lastLikePostID)
	}
}

func TestLikePostHandler_InvalidPostID(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/not-a-number/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUnlikePostHandler_NotLiked(t *testing.T) {
	uc := &fakePostUseCase{unlikeErr: entity.ErrNotLiked}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/post/42/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastUnlikePost != 42 {
		t.Fatalf("expected post_id 42, got %d", uc.lastUnlikePost)
	}
}

func TestLikePostHandler_PostNotFound(t *testing.T) {
	uc := &fakePostUseCase{likeErr: entity.ErrPostNotFound}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/42/like", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPostHandler_Success(t *testing.T) {
	uc := &fakePostUseCase{getByIDResp: response.PostDetail{PostID: 42, Username: "bob"}}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPostHandler_InvalidPostID(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetPostHandler_NotFound(t *testing.T) {
	uc := &fakePostUseCase{getByIDErr: entity.ErrPostNotFound}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPostHandler_Success(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/post/42", strings.NewReader(`{"caption":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastEditPID != 42 || uc.lastEditCaption != "updated" {
		t.Fatalf("unexpected edit call: post_id=%d caption=%q", uc.lastEditPID, uc.lastEditCaption)
	}
}

func TestEditPostHandler_Forbidden(t *testing.T) {
	uc := &fakePostUseCase{editErr: entity.ErrForbidden}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/post/42", strings.NewReader(`{"caption":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPostHandler_NotFound(t *testing.T) {
	uc := &fakePostUseCase{editErr: entity.ErrPostNotFound}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/post/42", strings.NewReader(`{"caption":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestEditPostHandler_InvalidPostID(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodPut, "/api/v1/post/abc", strings.NewReader(`{"caption":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePostHandler_Success(t *testing.T) {
	uc := &fakePostUseCase{}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/post/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastDeletePID != 42 {
		t.Fatalf("expected post_id 42, got %d", uc.lastDeletePID)
	}
}

func TestDeletePostHandler_Forbidden(t *testing.T) {
	uc := &fakePostUseCase{deleteErr: entity.ErrForbidden}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/post/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeletePostHandler_NotFound(t *testing.T) {
	uc := &fakePostUseCase{deleteErr: entity.ErrPostNotFound}
	router, st := newTestPostHandler(uc)
	defer os.RemoveAll(st.FullPath(""))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/post/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}
