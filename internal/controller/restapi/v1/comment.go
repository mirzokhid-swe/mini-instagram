package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
)

type createCommentRequest struct {
	Content string `json:"content"`
}

// createComment godoc
//
//	@Summary		Comment on a post
//	@Description	Adds a comment (1-2048 chars after trimming) to a post.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id	path		int					true	"Post ID"
//	@Param			request	body		createCommentRequest	true	"Comment content"
//	@Success		200		{object}	http.Response
//	@Failure		400		{object}	http.Response	"invalid post_id or empty content"
//	@Failure		401		{object}	http.Response
//	@Failure		404		{object}	http.Response	"post not found"
//	@Router			/post/{post_id}/comments [post]
func (h *V1) createComment(c *gin.Context) {
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

	var body createCommentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid request body")
		return
	}

	if err := h.comments.Create(c.Request.Context(), callerID, postID, body.Content); err != nil {
		h.handleUsecaseError(c, err, "create comment failed", "user_id", callerID, "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

// listComments godoc
//
//	@Summary		List comments on a post
//	@Description	Paginated list of a post's comments, oldest first, excluding soft-deleted comments.
//	@Tags			comments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			post_id		path		int	true	"Post ID"
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.CommentList}
//	@Failure		400			{object}	http.Response	"invalid post_id"
//	@Failure		401			{object}	http.Response
//	@Failure		404			{object}	http.Response	"post not found"
//	@Router			/post/{post_id}/comments [get]
func (h *V1) listComments(c *gin.Context) {
	_, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	postID, err := strconv.ParseInt(c.Param("post_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid post_id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	comments, err := h.comments.List(c.Request.Context(), postID, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "list comments failed", "post_id", postID)
		return
	}

	h.handleResponse(c, apihttp.OK, comments)
}

// deleteComment godoc
//
//	@Summary		Delete a comment
//	@Description	Soft-deletes a comment. Allowed for the comment author or the post owner.
//	@Tags			comments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			comment_id	path		int	true	"Comment ID"
//	@Success		200			{object}	http.Response
//	@Failure		400			{object}	http.Response	"invalid comment_id"
//	@Failure		401			{object}	http.Response
//	@Failure		403			{object}	http.Response	"not the comment author or post owner"
//	@Failure		404			{object}	http.Response	"comment not found"
//	@Router			/comments/{comment_id} [delete]
func (h *V1) deleteComment(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid comment_id")
		return
	}

	if err := h.comments.Delete(c.Request.Context(), callerID, commentID); err != nil {
		h.handleUsecaseError(c, err, "delete comment failed", "user_id", callerID, "comment_id", commentID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}

type editCommentRequest struct {
	Content string `json:"content"`
}

// editComment godoc
//
//	@Summary		Edit a comment
//	@Description	Updates a comment's content. Allowed for the comment author or the post owner.
//	@Tags			comments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			comment_id	path		int					true	"Comment ID"
//	@Param			request		body		editCommentRequest	true	"Updated content"
//	@Success		200			{object}	http.Response
//	@Failure		400			{object}	http.Response	"invalid comment_id, invalid request body, or empty content"
//	@Failure		401			{object}	http.Response
//	@Failure		403			{object}	http.Response	"not the comment author or post owner"
//	@Failure		404			{object}	http.Response	"comment not found"
//	@Router			/comments/{comment_id} [put]
func (h *V1) editComment(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	commentID, err := strconv.ParseInt(c.Param("comment_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid comment_id")
		return
	}

	var body editCommentRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid request body")
		return
	}

	if err := h.comments.Edit(c.Request.Context(), callerID, commentID, body.Content); err != nil {
		h.handleUsecaseError(c, err, "edit comment failed", "user_id", callerID, "comment_id", commentID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}
