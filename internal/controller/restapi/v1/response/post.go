package response

import "time"

// PostDetail is the response shape for GET /post/:post_id.
type PostDetail struct {
	PostID        int64     `json:"post_id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	Caption       string    `json:"caption"`
	ImagePath     string    `json:"image_path"`
	LikesCount    int64     `json:"likes_count"`
	CommentsCount int64     `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
	IsLiked       bool      `json:"is_liked"`
}
