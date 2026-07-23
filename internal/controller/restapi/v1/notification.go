package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
)

// listNotifications godoc
//
//	@Summary		List notifications
//	@Description	Paginated list of the caller's notifications (likes, comments, follows), newest first.
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page		query		int	false	"Page number (default 1)"
//	@Param			per_page	query		int	false	"Items per page (default 10, max 100)"
//	@Success		200			{object}	http.Response{data=response.NotificationList}
//	@Failure		401			{object}	http.Response
//	@Router			/notifications [get]
func (h *V1) listNotifications(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "10"))

	notifications, err := h.notifications.List(c.Request.Context(), callerID, page, perPage)
	if err != nil {
		h.handleUsecaseError(c, err, "list notifications failed", "user_id", callerID)
		return
	}

	h.handleResponse(c, apihttp.OK, notifications)
}

// markNotificationRead godoc
//
//	@Summary		Mark a notification as read
//	@Description	Idempotent; a notification that isn't the caller's own returns 404 (not 403) to avoid leaking existence.
//	@Tags			notifications
//	@Produce		json
//	@Security		BearerAuth
//	@Param			notification_id	path		int	true	"Notification ID"
//	@Success		200				{object}	http.Response
//	@Failure		400				{object}	http.Response	"invalid notification_id"
//	@Failure		401				{object}	http.Response
//	@Failure		404				{object}	http.Response	"notification not found"
//	@Router			/notifications/{notification_id}/read [put]
func (h *V1) markNotificationRead(c *gin.Context) {
	callerID, ok := currentUserID(c)
	if !ok {
		h.handleError(c, apihttp.Unauthorized, "unauthorized")
		return
	}

	notificationID, err := strconv.ParseInt(c.Param("notification_id"), 10, 64)
	if err != nil {
		h.handleError(c, apihttp.BadRequest, "invalid notification_id")
		return
	}

	if err := h.notifications.MarkRead(c.Request.Context(), notificationID, callerID); err != nil {
		h.handleUsecaseError(c, err, "mark notification read failed", "user_id", callerID, "notification_id", notificationID)
		return
	}

	h.handleResponse(c, apihttp.OK, nil)
}
