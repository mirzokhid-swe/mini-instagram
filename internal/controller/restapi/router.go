// Package restapi implements HTTP router.
package restapi

import (
	"github.com/gin-gonic/gin"

	v1 "mini-instagram/internal/controller/restapi/v1"
	"mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/usecase"
	jwtmanager "mini-instagram/pkg/jwt"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/redis"
	"mini-instagram/pkg/storage"
)

func NewRouter(handler *gin.Engine, auth usecase.Auth, posts usecase.Post, comments usecase.Comment, users usecase.User, notifications usecase.Notification, tokens *jwtmanager.TokenManager, l logger.Interface, st *storage.Storage, redisClient *redis.Client) {
	handler.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, http.Response{
			Status:      "OK",
			Description: "Service is running",
			Data:        "OK",
		})
	})

	handler.Static("/media", st.FullPath(""))

	api := handler.Group("/api/v1")
	{
		v1.NewRoutes(api, auth, posts, comments, users, notifications, tokens, l, st, redisClient)
	}
}
