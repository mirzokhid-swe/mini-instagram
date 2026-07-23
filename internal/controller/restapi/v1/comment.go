package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
)

type createCommentRequest struct {
	Content string `json:"content"`
}

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
