package v1

import (
	"errors"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
	"mini-instagram/internal/entity"
)

// handleUsecaseError maps a usecase error to an HTTP response. Errors the
// API contract knows about (bad input, conflicts, missing resources, bad
// credentials) are reported to the client with their real message.
// Anything else is logged with logMsg/logFields and returned as an opaque
// 500, so unexpected internal errors (DB failures, storage errors, ...)
// never leak their details to the caller.
func (h *V1) handleUsecaseError(c *gin.Context, err error, logMsg string, logFields ...any) {
	var vErr *entity.ValidationError

	switch {
	case errors.As(err, &vErr):
		h.handleError(c, apihttp.BadRequest, vErr.Message)
	case errors.Is(err, entity.ErrNotFound):
		h.handleError(c, apihttp.NOT_FOUND, "user not found")
	case errors.Is(err, entity.ErrPostNotFound):
		h.handleError(c, apihttp.NOT_FOUND, "post not found")
	case errors.Is(err, entity.ErrUsernameTaken):
		h.handleError(c, apihttp.Conflict, "username already exists")
	case errors.Is(err, entity.ErrEmailTaken):
		h.handleError(c, apihttp.Conflict, "email already exists")
	case errors.Is(err, entity.ErrInvalidCredentials):
		h.handleError(c, apihttp.Unauthorized, "invalid email or password")
	case errors.Is(err, entity.ErrNotLiked):
		h.handleError(c, apihttp.Conflict, "post is not liked")
	case errors.Is(err, entity.ErrCommentNotFound):
		h.handleError(c, apihttp.NOT_FOUND, "comment not found")
	case errors.Is(err, entity.ErrForbidden):
		h.handleError(c, apihttp.Forbidden, "forbidden")
	case errors.Is(err, entity.ErrSelfFollow):
		h.handleError(c, apihttp.BadRequest, "cannot follow yourself")
	case errors.Is(err, entity.ErrAlreadyFollowing):
		h.handleError(c, apihttp.Conflict, "already following")
	case errors.Is(err, entity.ErrNotFollowing):
		h.handleError(c, apihttp.Conflict, "not following")
	default:
		h.logger.Error(logMsg, append(logFields, "error", err)...)
		h.handleError(c, apihttp.InternalServerError, "internal server error")
	}
}
