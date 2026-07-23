// Package v1 implements HTTP routes for API version 1.
package v1

import (
	"github.com/gin-gonic/gin"

	"mini-instagram/internal/controller/restapi/middleware"
	"mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/usecase"
	jwtmanager "mini-instagram/pkg/jwt"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/redis"
	"mini-instagram/pkg/storage"
)

// V1 -.
type V1 struct {
	auth    usecase.Auth
	posts   usecase.Post
	users   usecase.User
	logger  logger.Interface
	storage *storage.Storage
	redis   *redis.Client
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
func NewRoutes(api *gin.RouterGroup, auth usecase.Auth, posts usecase.Post, users usecase.User, tokens *jwtmanager.TokenManager, l logger.Interface, st *storage.Storage, redisClient *redis.Client) {
	h := &V1{auth: auth, posts: posts, users: users, logger: l, storage: st, redis: redisClient}
	authRoutes := api.Group("/auth")
	authRoutes.POST("/sign-up", middleware.RateLimitByEmail(h.redis, "rl:signup:", h.logger), h.signUp)
	authRoutes.POST("/login", middleware.RateLimitByEmail(h.redis, "rl:login:", h.logger), h.login)
	authRoutes.POST("/logout", middleware.Auth(tokens), h.logout)

	protected := api.Group("/")
	protected.Use(middleware.Auth(tokens))
	{
		protected.GET("/profile", h.getProfile)

		protected.Group("post")
		{
			protected.POST("", h.createPost)
		}

		users := protected.Group("/users")
		{
			users.GET("/:user_id", h.getUserProfile)
			users.GET("/:user_id/posts", h.getUserPosts)
		}
	}
}
