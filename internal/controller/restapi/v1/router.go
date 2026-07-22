// Package v1 implements HTTP routes for API version 1.
package v1

import (
	"github.com/gin-gonic/gin"

	"mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/usecase"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

// V1 -.
type V1 struct {
	auth    usecase.Auth
	logger  logger.Interface
	storage *storage.Storage
}

func (h *V1) handleResponse(c *gin.Context, status http.Status, data interface{}) {
	c.JSON(status.Code, http.Response{
		Status:      status.Status,
		Description: status.Description,
		Data:        data,
	})
}

func (h *V1) handleError(c *gin.Context, status http.Status, message string) {
	c.JSON(status.Code, http.Response{
		Status:      status.Status,
		Description: message,
		Data:        nil,
	})
}

// NewRoutes -.
func NewRoutes(api *gin.RouterGroup, auth usecase.Auth, l logger.Interface, st *storage.Storage) {
	h := &V1{auth: auth, logger: l, storage: st}
	authRoutes := api.Group("/auth")
	authRoutes.POST("/sign-up", h.signUp)
	authRoutes.POST("/login", h.login)
}
