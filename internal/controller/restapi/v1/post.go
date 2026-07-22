package v1

import (
	"errors"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/pkg/image"
)

func (h *V1) createPost(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}
	uid, ok := userID.(int64)
	if !ok {
		h.logger.Error("user_id has invalid type")
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	if err := c.Request.ParseMultipartForm(image.DefaultMaxSize); err != nil {
		h.logger.Info("create post request parsing failed", "error", err)
		h.handleError(c, apihttp.BadRequest, "invalid multipart request")
		return
	}

	caption := c.Request.FormValue("caption")
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		h.logger.Info("create post missing image", "error", err)
		h.handleError(c, apihttp.BadRequest, "image is required")
		return
	}
	defer file.Close()

	if err := h.posts.Create(c.Request.Context(), request.CreatePost{UserID: uid, Caption: caption, File: file, Header: header}); err != nil {
		switch {
		case errors.Is(err, entity.ErrNotFound):
			h.handleError(c, apihttp.NOT_FOUND, "user not found")
		default:
			h.logger.Error("create post failed", "user_id", uid, "error", err)
			h.handleError(c, apihttp.BadRequest, err.Error())
		}
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}
