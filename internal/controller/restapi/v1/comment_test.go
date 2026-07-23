package v1

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/logger"
)

type fakeCommentUseCase struct {
	createErr    error
	lastPostID   int64
	lastContent  string
	listResp     response.CommentList
	listErr      error
	deleteErr    error
	lastDeleteID int64
}

func (f *fakeCommentUseCase) Create(ctx context.Context, callerID, postID int64, content string) error {
	f.lastPostID, f.lastContent = postID, content
	return f.createErr
}

func (f *fakeCommentUseCase) List(ctx context.Context, postID int64, page, perPage int) (response.CommentList, error) {
	return f.listResp, f.listErr
}

func (f *fakeCommentUseCase) Delete(ctx context.Context, callerID, commentID int64) error {
	f.lastDeleteID = commentID
	return f.deleteErr
}

func newTestCommentHandler(uc *fakeCommentUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := gin.New()
	h := &V1{comments: uc, logger: logger.New("error")}

	auth := handler.Group("/api/v1")
	auth.Use(func(c *gin.Context) {
		c.Set("user_id", int64(1))
	})
	{
		auth.POST("/post/:post_id/comments", h.createComment)
		auth.GET("/post/:post_id/comments", h.listComments)
		auth.DELETE("/comments/:comment_id", h.deleteComment)
	}
	return handler
}

func TestCreateCommentHandler_Success(t *testing.T) {
	uc := &fakeCommentUseCase{}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/42/comments", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastPostID != 42 || uc.lastContent != "hello" {
		t.Fatalf("unexpected create call: post_id=%d content=%q", uc.lastPostID, uc.lastContent)
	}
}

func TestCreateCommentHandler_InvalidPostID(t *testing.T) {
	uc := &fakeCommentUseCase{}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/abc/comments", strings.NewReader(`{"content":"hello"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCommentHandler_ValidationError(t *testing.T) {
	uc := &fakeCommentUseCase{createErr: entity.NewValidationError("content", "content is required")}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/42/comments", strings.NewReader(`{"content":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCommentHandler_PostNotFound(t *testing.T) {
	uc := &fakeCommentUseCase{createErr: entity.ErrPostNotFound}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/post/42/comments", strings.NewReader(`{"content":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListCommentsHandler_Success(t *testing.T) {
	uc := &fakeCommentUseCase{listResp: response.CommentList{Count: 1, Items: []response.CommentItem{{CommentID: 1}}}}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/post/42/comments", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCommentHandler_Success(t *testing.T) {
	uc := &fakeCommentUseCase{}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/7", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
	if uc.lastDeleteID != 7 {
		t.Fatalf("expected comment_id 7, got %d", uc.lastDeleteID)
	}
}

func TestDeleteCommentHandler_Forbidden(t *testing.T) {
	uc := &fakeCommentUseCase{deleteErr: entity.ErrForbidden}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/7", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCommentHandler_NotFound(t *testing.T) {
	uc := &fakeCommentUseCase{deleteErr: entity.ErrCommentNotFound}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/7", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteCommentHandler_InvalidID(t *testing.T) {
	uc := &fakeCommentUseCase{}
	router := newTestCommentHandler(uc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/abc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
