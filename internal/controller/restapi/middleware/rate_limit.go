package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/pkg/logger"
	"mini-instagram/pkg/redis"
)

const (
	rateLimitMaxAttempts = 5
	rateLimitWindow      = time.Minute
)

type emailRequest struct {
	Email string `json:"email"`
}

func RateLimitByEmail(client *redis.Client, prefix string, l logger.Interface) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := prefix + requestEmail(c.Request)
		if client == nil {
			c.Next()
			return
		}

		count, err := client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			l.Error("rate limit increment failed", "key", key, "error", err)
			c.Next()
			return
		}
		if count == 1 {
			if err := client.Expire(c.Request.Context(), key, rateLimitWindow).Err(); err != nil {
				l.Error("rate limit expiry failed", "key", key, "error", err)
			}
		}
		if count > rateLimitMaxAttempts {
			c.JSON(apihttp.TooManyRequests.Code, apihttp.Response{
				Status:      apihttp.TooManyRequests.Status,
				Description: "too many attempts, try again later",
				Data:        nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

func requestEmail(r *http.Request) string {
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var email string

	switch contentType {
	case "application/json":
		body, err := io.ReadAll(r.Body)
		if err == nil {
			r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))
			var request emailRequest
			if json.Unmarshal(body, &request) == nil {
				email = request.Email
			}
		}
	default:
		if err := r.ParseMultipartForm(32 << 20); err == nil {
			email = r.FormValue("email")
		}
	}

	email = strings.TrimSpace(strings.ToLower(email))
	if email != "" {
		return email
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
