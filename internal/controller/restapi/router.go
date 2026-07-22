// Package restapi implements HTTP router.
package restapi

import (
	"github.com/gin-gonic/gin"

	v1 "mini-instagram/internal/controller/restapi/v1"
	"mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/usecase"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/storage"
)

func NewRouter(handler *gin.Engine, auth usecase.Auth, l logger.Interface, st *storage.Storage) {
	handler.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, http.Response{
			Status:      "OK",
			Description: "Service is running",
			Data:        "OK",
		})
	})

	api := handler.Group("/api/v1")
	{
		v1.NewRoutes(api, auth, l, st)
	}
}
