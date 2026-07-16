// Package v1 implements HTTP routes for API version 1.
package v1

import (
	"github.com/gin-gonic/gin"

	"todo/internal/controller/restapi/v1/http"
)

// V1 -.
type V1 struct {
}

func (h *V1) handleResponse(c *gin.Context, status http.Status, data interface{}) {
	c.JSON(status.Code, http.Response{
		Status:      status.Status,
		Description: status.Description,
		Data:        data,
	})
}

// NewRoutes -.
func NewRoutes(api *gin.RouterGroup) {
	_ = api
	_ = &V1{}
}
