package notification

import (
	"context"
	"fmt"

	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 10
	MaxPerPage     = 100
)

type UseCase struct {
	notifications repo.Notification
}

func New(notifications repo.Notification) usecase.Notification {
	return &UseCase{notifications: notifications}
}

func (u *UseCase) List(ctx context.Context, userID int64, page, perPage int) (response.NotificationList, error) {
	if page < 1 {
		page = DefaultPage
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	offset := (page - 1) * perPage

	count, err := u.notifications.Count(ctx, userID)
	if err != nil {
		return response.NotificationList{}, fmt.Errorf("count notifications: %w", err)
	}

	notifications, err := u.notifications.List(ctx, userID, perPage, offset)
	if err != nil {
		return response.NotificationList{}, fmt.Errorf("list notifications: %w", err)
	}

	items := make([]response.NotificationItem, len(notifications))
	for i, n := range notifications {
		items[i] = response.NotificationItem{
			NotificationID: n.ID,
			ActionType:     n.ActionType,
			ActorID:        n.ActorID,
			ActorUsername:  n.ActorUsername,
			PostID:         n.PostID,
			IsRead:         n.IsRead,
			CreatedAt:      n.CreatedAt,
		}
	}

	return response.NotificationList{Count: count, Items: items}, nil
}

func (u *UseCase) MarkRead(ctx context.Context, notificationID, userID int64) error {
	if err := u.notifications.MarkRead(ctx, notificationID, userID); err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}
