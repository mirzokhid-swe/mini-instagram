package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	jwtmanager "mini-instagram/pkg/jwt"
)

// Auth validates a JWT access token and stores the user_id in the gin context.
func Auth(tokens *jwtmanager.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.JSON(http.StatusUnauthorized, apihttp.Response{
				Status:      apihttp.Unauthorized.Status,
				Description: "missing authorization header",
				Data:        nil,
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" || parts[1] == "" {
			c.JSON(http.StatusUnauthorized, apihttp.Response{
				Status:      apihttp.Unauthorized.Status,
				Description: "invalid authorization header",
				Data:        nil,
			})
			c.Abort()
			return
		}

		claims, err := tokens.ValidateToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, apihttp.Response{
				Status:      apihttp.Unauthorized.Status,
				Description: "invalid or expired token",
				Data:        nil,
			})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}
