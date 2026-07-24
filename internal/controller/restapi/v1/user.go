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

// getProfile godoc
//
//	@Summary		Get own profile
//	@Description	Returns the authenticated caller's profile.
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	http.Response{data=response.Profile}
//	@Failure		401	{object}	http.Response
//	@Router			/profile [get]
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

// getUserProfile godoc
//
//	@Summary		Get a user's profile
//	@Description	Returns the given user's profile, including whether the caller follows them.
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		int	true	"User ID"
//	@Success		200		{object}	http.Response{data=response.Profile}
//	@Failure		400		{object}	http.Response	"invalid user_id"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"user not found"
//	@Router			/users/{user_id} [get]
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

// getUserPosts godoc
//
//	@Summary		List a user's posts
//	@Description	Paginated list of a user's posts, newest first, excluding soft-deleted posts.
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		int	true	"User ID"
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.UserPosts}
//	@Failure		400			{object}	http.Response	"invalid user_id"
//	@Failure		401			{object}	http.Response
//	@Router			/users/{user_id}/posts [get]
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

// listFollowers godoc
//
//	@Summary		List a user's followers
//	@Description	Paginated list of accounts that follow the given user.
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		int	true	"User ID"
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.UserSearch}
//	@Failure		400			{object}	http.Response	"invalid user_id"
//	@Failure		401			{object}	http.Response
//	@Failure		404			{object}	http.Response	"user not found"
//	@Router			/users/{user_id}/followers [get]
func (h *V1) listFollowers(c *gin.Context) {
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	results, err := h.users.ListFollowers(c.Request.Context(), callerID, userID, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "list followers failed", "user_id", userID)
		return
	}

	h.handleResponse(c, apihttp.OK, results)
}

// listFollowing godoc
//
//	@Summary		List who a user follows
//	@Description	Paginated list of accounts the given user follows.
//	@Tags			profile
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id		path		int	true	"User ID"
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.UserSearch}
//	@Failure		400			{object}	http.Response	"invalid user_id"
//	@Failure		401			{object}	http.Response
//	@Failure		404			{object}	http.Response	"user not found"
//	@Router			/users/{user_id}/following [get]
func (h *V1) listFollowing(c *gin.Context) {
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	results, err := h.users.ListFollowing(c.Request.Context(), callerID, userID, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "list following failed", "user_id", userID)
		return
	}

	h.handleResponse(c, apihttp.OK, results)
}

// editProfile godoc
//
//	@Summary		Edit own profile
//	@Description	Updates the caller's username, full name, bio, and optionally the avatar (max 5 MB, jpeg/png/webp).
//	@Tags			profile
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			username	formData	string	true	"Username"
//	@Param			full_name	formData	string	true	"Full name"
//	@Param			bio			formData	string	false	"Bio"
//	@Param			avatar		formData	file	false	"New avatar image (jpeg/png/webp, max 5MB)"
//	@Success		200			{object}	http.Response
//	@Failure		400			{object}	http.Response	"validation error or avatar too large"
//	@Failure		401			{object}	http.Response
//	@Failure		409			{object}	http.Response	"username already taken"
//	@Router			/profile [put]
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
		h.handleFieldError(c, apihttp.BadRequest, "avatar", "invalid avatar upload")
		return
	}

	if err := h.users.UpdateProfile(c.Request.Context(), input); err != nil {
		h.handleUsecaseError(c, err, "edit profile failed", "user_id", callerID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// followUser godoc
//
//	@Summary		Follow a user
//	@Description	Follows the given user. Following yourself, a missing/inactive user, or an already-followed user are rejected.
//	@Tags			follow
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		int	true	"User ID to follow"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"cannot follow yourself"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"user not found"
//	@Failure		409		{object}	http.Response	"already following"
//	@Router			/users/{user_id}/follow [post]
func (h *V1) followUser(c *gin.Context) {
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

	if err := h.users.Follow(c.Request.Context(), callerID, userID); err != nil {
		h.handleUsecaseError(c, err, "follow user failed", "follower_id", callerID, "following_id", userID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// unfollowUser godoc
//
//	@Summary		Unfollow a user
//	@Description	Removes an existing follow relationship.
//	@Tags			follow
//	@Produce		json
//	@Security		BearerAuth
//	@Param			user_id	path		int	true	"User ID to unfollow"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"invalid user_id"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"user not found"
//	@Failure		409		{object}	http.Response	"not following"
//	@Router			/users/{user_id}/follow [delete]
func (h *V1) unfollowUser(c *gin.Context) {
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

	if err := h.users.Unfollow(c.Request.Context(), callerID, userID); err != nil {
		h.handleUsecaseError(c, err, "unfollow user failed", "follower_id", callerID, "following_id", userID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// searchUsers godoc
//
//	@Summary		Search users
//	@Description	Case-insensitive substring search on username, exact/prefix matches ranked first.
//	@Tags			search
//	@Produce		json
//	@Security		BearerAuth
//	@Param			q			query		string	true	"Search term (1-32 chars)"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.UserSearch}
//	@Failure		400			{object}	http.Response	"missing or invalid q"
//	@Failure		401			{object}	http.Response
//	@Router			/search/users [get]
func (h *V1) searchUsers(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	query := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	results, err := h.users.SearchUsers(c.Request.Context(), callerID, query, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "search users failed", "query", query)
		return
	}

	h.handleResponse(c, apihttp.OK, results)
}

func currentUserID(c *gin.Context) (int64, bool) {
	v, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	uid, ok := v.(int64)
	return uid, ok
}
