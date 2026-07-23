package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/pkg/image"
)

// createPost godoc
//
//	@Summary		Create a post
//	@Description	Uploads an image (max 10 MB, jpeg/png/webp), generates a thumbnail, and parses hashtags from the caption.
//	@Tags			posts
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			image	formData	file	true	"Post image (jpeg/png/webp, max 10MB)"
//	@Param			caption	formData	string	false	"Caption (max 2048 chars, may include #hashtags)"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"missing/invalid image or caption too long"
//	@Failure		401		{object}	http.Response
//	@Router			/post [post]
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

// likePost godoc
//
//	@Summary		Like a post
//	@Description	Idempotent: liking an already-liked post returns 200 without changing the like count.
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		int	true	"Post ID"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"invalid post_id"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"post not found"
//	@Router			/post/{post_id}/like [post]
func (h *V1) likePost(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid post_id")
		return
	}

	if err := h.posts.Like(c.Request.Context(), callerID, postID); err != nil {
		h.handleUsecaseError(c, err, "like post failed", "user_id", callerID, "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// unlikePost godoc
//
//	@Summary		Unlike a post
//	@Description	Removes a like. Returns 409 if the caller hadn't liked the post.
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		int	true	"Post ID"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"invalid post_id"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"post not found"
//	@Failure		409		{object}	http.Response	"post is not liked"
//	@Router			/post/{post_id}/like [delete]
func (h *V1) unlikePost(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid post_id")
		return
	}

	if err := h.posts.Unlike(c.Request.Context(), callerID, postID); err != nil {
		h.handleUsecaseError(c, err, "unlike post failed", "user_id", callerID, "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// searchPostsByTag godoc
//
//	@Summary		Search posts by hashtag
//	@Description	Returns posts tagged with the given hashtag (leading # is optional), newest first. Unknown tags return an empty list.
//	@Tags			search
//	@Produce		json
//	@Security		BearerAuth
//	@Param			tag			query		string	true	"Hashtag name, with or without leading #"
//	@Param			page		query		int		false	"Page number (default 1)"
//	@Param			per_page	query		int		false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.HashtagPostList}
//	@Failure		400			{object}	http.Response	"missing tag"
//	@Failure		401			{object}	http.Response
//	@Router			/search/posts [get]
func (h *V1) searchPostsByTag(c *gin.Context) {
	_, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	tag := c.Query("tag")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	results, err := h.posts.SearchByTag(c.Request.Context(), tag, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "search posts by tag failed", "tag", tag)
		return
	}

	h.handleResponse(c, apihttp.OK, results)
}

// getPost godoc
//
//	@Summary		Get a single post
//	@Description	Returns post details including like/comment counts and whether the caller has liked it.
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		int	true	"Post ID"
//	@Success		200		{object}	http.Response{data=response.PostDetail}
//	@Failure		400		{object}	http.Response	"invalid post_id"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"post not found"
//	@Router			/post/{post_id} [get]
func (h *V1) getPost(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid post_id")
		return
	}

	post, err := h.posts.GetByID(c.Request.Context(), callerID, postID)
	if err != nil {
		h.handleUsecaseError(c, err, "get post failed", "user_id", callerID, "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, post)
}

// deletePost godoc
//
//	@Summary		Delete a post
//	@Description	Soft-deletes the post and removes its image/thumbnail files. Only the post owner may delete it.
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		int	true	"Post ID"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"invalid post_id"
//	@Failure		401		{object}	http.Response
//	@Failure		403		{object}	http.Response	"not the post owner"
//	@Failure		404		{object}	http.Response	"post not found"
//	@Router			/post/{post_id} [delete]
func (h *V1) deletePost(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid post_id")
		return
	}

	if err := h.posts.Delete(c.Request.Context(), callerID, postID); err != nil {
		h.handleUsecaseError(c, err, "delete post failed", "user_id", callerID, "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// getFeed godoc
//
//	@Summary		Home feed
//	@Description	Posts authored by users the caller follows (not the caller's own posts), newest first.
//	@Tags			posts
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.Feed}
//	@Failure		401			{object}	http.Response
//	@Router			/feed [get]
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
