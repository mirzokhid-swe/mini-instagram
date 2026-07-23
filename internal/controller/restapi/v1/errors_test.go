package v1

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"mini-instagram/internal/entity"
)

type nopLogger struct{}

func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}

func newTestV1() *V1 {
	return &V1{logger: nopLogger{}}
}

func TestHandleUsecaseError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "validation error",
			err:        entity.NewValidationError("username", "username must be between 3 and 32 characters"),
			wantStatus: 400,
			wantBody:   "username must be between 3 and 32 characters",
		},
		{
			name:       "not found",
			err:        entity.ErrNotFound,
			wantStatus: 404,
			wantBody:   "user not found",
		},
		{
			name:       "username taken",
			err:        entity.ErrUsernameTaken,
			wantStatus: 409,
			wantBody:   "username already exists",
		},
		{
			name:       "email taken",
			err:        entity.ErrEmailTaken,
			wantStatus: 409,
			wantBody:   "email already exists",
		},
		{
			name:       "invalid credentials",
			err:        entity.ErrInvalidCredentials,
			wantStatus: 401,
			wantBody:   "invalid email or password",
		},
		{
			name:       "unknown error is opaque",
			err:        errors.New("pq: connection refused"),
			wantStatus: 500,
			wantBody:   "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			h := newTestV1()
			h.handleUsecaseError(c, tt.err, "test failed")

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.wantBody) {
				t.Fatalf("expected body to contain %q, got %s", tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestHandleUsecaseError_DoesNotLeakUnknownErrorDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h := newTestV1()
	h.handleUsecaseError(c, errors.New("save avatar: disk full at /var/data/secret-path"), "edit profile failed")

	if strings.Contains(w.Body.String(), "secret-path") {
		t.Fatalf("expected internal error details not to be exposed to the client, got %s", w.Body.String())
	}
	if w.Code != 500 {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}
