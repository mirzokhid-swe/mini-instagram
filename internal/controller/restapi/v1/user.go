package v1

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/pkg/image"
)

func (h *V1) getProfile(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	profile, err := h.users.GetProfile(c.Request.Context(), callerID, callerID)
	if err != nil {
		h.handleUsecaseError(c, err, "get profile failed", "user_id", callerID)
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
		h.handleUsecaseError(c, err, "get profile failed", "user_id", userID)
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
		h.handleUsecaseError(c, err, "get user posts failed", "user_id", userID)
		return
	}

	h.handleResponse(c, apihttp.OK, posts)
}

func (h *V1) editProfile(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	if err := c.Request.ParseMultipartForm(image.DefaultMaxSize); err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid multipart request")
		return
	}

	username := strings.TrimSpace(strings.ToLower(c.Request.FormValue("username")))
	fullName := strings.TrimSpace(c.Request.FormValue("full_name"))
	bio := strings.TrimSpace(c.Request.FormValue("bio"))

	input := request.UpdateProfile{
		UserID:   callerID,
		Username: username,
		FullName: fullName,
		Bio:      bio,
	}

	file, header, err := c.Request.FormFile("avatar")
	if err == nil {
		defer file.Close()
		input.Avatar = file
		input.AvatarHeader = header
	} else if !errors.Is(err, http.ErrMissingFile) {
		h.handleError(c, apihttp.BadRequest, "invalid avatar upload")
		return
	}

	if err := h.users.UpdateProfile(c.Request.Context(), input); err != nil {
		h.handleUsecaseError(c, err, "edit profile failed", "user_id", callerID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

func currentUserID(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}
