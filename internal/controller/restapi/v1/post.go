package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
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
		h.handleUsecaseError(c, err, "create post failed", "user_id", uid)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

func (h *V1) getFeed(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	feed, err := h.posts.GetFeed(c.Request.Context(), callerID, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "get feed failed", "user_id", callerID)
		return
	}

	h.handleResponse(c, apihttp.OK, feed)
}
