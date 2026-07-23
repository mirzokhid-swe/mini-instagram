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
	auth     usecase.Auth
	posts    usecase.Post
	comments usecase.Comment
	users    usecase.User
	logger   logger.Interface
	storage  *storage.Storage
	redis    *redis.Client
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
func NewRoutes(api *gin.RouterGroup, auth usecase.Auth, posts usecase.Post, comments usecase.Comment, users usecase.User, tokens *jwtmanager.TokenManager, l logger.Interface, st *storage.Storage, redisClient *redis.Client) {
	h := &V1{auth: auth, posts: posts, comments: comments, users: users, logger: l, storage: st, redis: redisClient}
	authRoutes := api.Group("/auth")
	authRoutes.POST("/sign-up", middleware.RateLimitByEmail(h.redis, "rl:signup:", h.logger), h.signUp)
	authRoutes.POST("/login", middleware.RateLimitByEmail(h.redis, "rl:login:", h.logger), h.login)
	authRoutes.POST("/logout", middleware.Auth(tokens), h.logout)

	protected := api.Group("/")
	protected.Use(middleware.Auth(tokens))
	{
		profile := protected.Group("/profile")
		{
			profile.GET("", h.getProfile)
			profile.PUT("", h.editProfile)
		}

		postRoutes := protected.Group("/post")
		{
			postRoutes.POST("", h.createPost)
			postRoutes.GET("/:post_id", h.getPost)
			postRoutes.DELETE("/:post_id", h.deletePost)
			postRoutes.POST("/:post_id/like", h.likePost)
			postRoutes.DELETE("/:post_id/like", h.unlikePost)
			postRoutes.POST("/:post_id/comments", h.createComment)
			postRoutes.GET("/:post_id/comments", h.listComments)
		}

		protected.DELETE("/comments/:comment_id", h.deleteComment)

		protected.GET("/feed", h.getFeed)

		users := protected.Group("/users")
		{
			users.GET("/:user_id", h.getUserProfile)
			users.GET("/:user_id/posts", h.getUserPosts)
			users.POST("/:user_id/follow", h.followUser)
			users.DELETE("/:user_id/follow", h.unfollowUser)
		}
	}
}
