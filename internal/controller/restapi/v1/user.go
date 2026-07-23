package v1

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/entity"
)

func (h *V1) getProfile(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	profile, err := h.users.GetProfile(c.Request.Context(), callerID, callerID)
	if err != nil {
		h.handleUserProfileError(c, err)
		return
	}

	h.handleResponse(c, apihttp.OK, profile)
}

func (h *V1) getUserProfile(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid user_id")
		return
	}

	profile, err := h.users.GetProfile(c.Request.Context(), userID, callerID)
	if err != nil {
		h.handleUserProfileError(c, err)
		return
	}

	h.handleResponse(c, apihttp.OK, profile)
}

func (h *V1) getUserPosts(c *gin.Context) {
	_, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid user_id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	posts, err := h.users.GetUserPosts(c.Request.Context(), userID, page, perPage)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			h.handleError(c, apihttp.NOT_FOUND, "user not found")
			return
		}
		h.logger.Error("get user posts failed", "user_id", userID, "error", err)
		h.handleError(c, apihttp.InternalServerError, "could not get user posts")
		return
	}

	h.handleResponse(c, apihttp.OK, posts)
}

func (h *V1) handleUserProfileError(c *gin.Context, err error) {
	if errors.Is(err, entity.ErrNotFound) {
		h.handleError(c, apihttp.NOT_FOUND, "user not found")
		return
	}
	h.logger.Error("get profile failed", "error", err)
	h.handleError(c, apihttp.InternalServerError, "could not get profile")
}

func currentUserID(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}
