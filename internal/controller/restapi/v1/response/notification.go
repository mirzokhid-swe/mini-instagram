package response

import "time"

// NotificationItem is a single notification entry in a user's notification list.
type NotificationItem struct {
	NotificationID int64     `json:"notification_id"`
	ActionType     string    `json:"action_type"`
	ActorID        int64     `json:"actor_id"`
	ActorUsername  string    `json:"actor_username"`
	PostID         int64     `json:"post_id"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}

// NotificationList is the paginated response for GET /notifications.
type NotificationList struct {
	Count int64              `json:"count"`
	Items []NotificationItem `json:"items"`
}
