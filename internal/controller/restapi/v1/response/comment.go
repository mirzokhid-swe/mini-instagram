package response

import "time"

// CommentItem is a single comment entry in a post's comment list.
type CommentItem struct {
	CommentID int64     `json:"comment_id"`
	PostID    int64     `json:"post_id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentList is the paginated response for GET /post/:post_id/comments.
type CommentList struct {
	Count int64         `json:"count"`
	Items []CommentItem `json:"items"`
}
