package entity

import "time"

type Post struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	ImagePath     string    `json:"image_path"`
	ThumbnailPath string    `json:"thumbnail_path"`
	Caption       string    `json:"caption"`
	LikeCount     int64     `json:"likes_count"`
	CommentCount  int64     `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
