package response

import "time"

// Profile is the response shape for GET /profile and GET /users/:user_id.
type Profile struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	FullName       string `json:"full_name"`
	Bio            string `json:"bio"`
	AvatarPath     string `json:"avatar_path"`
	PostsCount     int64  `json:"posts_count"`
	FollowersCount int64  `json:"followers_count"`
	FollowingCount int64  `json:"following_count"`
	IsFollowing    bool   `json:"is_following"`
}

// PostItem is a lightweight post returned in user post listings.
type PostItem struct {
	PostID        int64     `json:"post_id"`
	ThumbnailPath string    `json:"thumbnail_path"`
	Caption       string    `json:"caption"`
	LikesCount    int64     `json:"likes_count"`
	CommentsCount int64     `json:"comments_count"`
	CreatedAt     time.Time `json:"created_at"`
}

// UserPosts is the paginated response for GET /users/:user_id/posts.
type UserPosts struct {
	Count int64      `json:"count"`
	Items []PostItem `json:"items"`
}

// UserSearchItem is a single user entry in a username search result.
type UserSearchItem struct {
	UserID      int64  `json:"user_id"`
	Username    string `json:"username"`
	FullName    string `json:"full_name"`
	AvatarPath  string `json:"avatar_path"`
	IsFollowing bool   `json:"is_following"`
}

// UserSearch is the paginated response for GET /search/users.
type UserSearch struct {
	Count int64            `json:"count"`
	Items []UserSearchItem `json:"items"`
}
