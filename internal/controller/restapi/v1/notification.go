package v1

import (
	"strconv"

	"github.com/gin-gonic/gin"

	apihttp "mini-instagram/internal/controller/restapi/v1/http"
)

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
