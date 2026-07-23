package entity

import "time"

type Comment struct {
	ID        int64
	PostID    int64
	UserID    int64
	Username  string
	Content   string
	CreatedAt time.Time
}

// CommentOwnership carries just enough to decide who may delete a comment:
// the comment author or the author of the post it belongs to.
type CommentOwnership struct {
	CommentID   int64
	PostID      int64
	AuthorID    int64
	PostOwnerID int64
}
