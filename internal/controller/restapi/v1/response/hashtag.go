package response

import "time"

// HashtagPostItem is a single post entry in a hashtag search result.
type HashtagPostItem struct {
	PostID        int64     `json:"post_id"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username"`
	ThumbnailPath string    `json:"thumbnail_path"`
	Caption       string    `json:"caption"`
	LikesCount    int64     `json:"likes_count"`
	CommentsCount int64     `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// HashtagPostList is the paginated response for GET /search/posts?tag=.
type HashtagPostList struct {
	Count int64             `json:"count"`
	Items []HashtagPostItem `json:"items"`
}
