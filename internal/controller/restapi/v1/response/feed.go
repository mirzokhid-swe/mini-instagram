package response

import "time"

// FeedItem is a single post entry in the home feed.
type FeedItem struct {
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	PostID        int64     `json:"post_id"`
	Caption       string    `json:"caption"`
	ImagePath     string    `json:"image_path"`
	LikesCount    int64     `json:"likes_count"`
	CommentsCount int64     `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// Feed is the paginated response for GET /feed.
type Feed struct {
	Count int64      `json:"count"`
	Items []FeedItem `json:"items"`
}
