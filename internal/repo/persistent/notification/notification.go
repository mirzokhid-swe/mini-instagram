package notification

import (
	"context"
	"fmt"

	"mini-instagram/internal/entity"
	"mini-instagram/pkg/postgres"
)

type NotificationRepo struct {
	pool *postgres.Postgres
}

func NewNotificationRepo(pg *postgres.Postgres) *NotificationRepo {
	return &NotificationRepo{pool: pg}
}

func (r *NotificationRepo) Count(ctx context.Context, userID int64) (int64, error) {
	const query = `SELECT COUNT(*) FROM notifications WHERE user_id = $1`

	var count int64
	if err := r.pool.Pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count notifications: %w", err)
	}
	return count, nil
}

func (r *NotificationRepo) List(ctx context.Context, userID int64, limit, offset int) ([]entity.Notification, error) {
	const query = `
		SELECT n.id, n.action_type, n.actor_id, u.username, COALESCE(n.post_id, 0), n.is_read, n.created_at
		FROM notifications n
		JOIN users u ON u.id = n.actor_id
		WHERE n.user_id = $1
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []entity.Notification
	for rows.Next() {
		var n entity.Notification
		if err := rows.Scan(&n.ID, &n.ActionType, &n.ActorID, &n.ActorUsername, &n.PostID, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan notification row: %w", err)
		}
		notifications = append(notifications, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification rows: %w", err)
	}

	return notifications, nil
}

func (r *NotificationRepo) MarkRead(ctx context.Context, notificationID, userID int64) error {
	tag, err := r.pool.Pool.Exec(ctx,
		`UPDATE notifications SET is_read = true, updated_at = now() WHERE id = $1 AND user_id = $2`,
		notificationID, userID,
	)
	if err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrNotificationNotFound
	}
	return nil
}
