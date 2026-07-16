// Package restapi implements HTTP router.
package restapi

import (
	"net/http"

	"github.com/gin-gonic/gin"

	v1 "todo/internal/controller/restapi/v1"
)

func NewRouter(handler *gin.Engine) {
	handler.GET("/healthz", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	api := handler.Group("/api/v1")
	{
		v1.NewRoutes(api)
	}
}
