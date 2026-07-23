package comment

import (
	"context"
	"fmt"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"mini-instagram/internal/controller/restapi/v1/response"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	"mini-instagram/internal/usecase"
)

const (
	MaxContentLength = 2048
	DefaultPage      = 1
	DefaultPerPage   = 10
	MaxPerPage       = 100
)

type UseCase struct {
	comments repo.Comment
}

func New(comments repo.Comment) usecase.Comment {
	return &UseCase{comments: comments}
}

func (u *UseCase) Create(ctx context.Context, callerID, postID int64, content string) error {
	content = strings.TrimSpace(bluemonday.StrictPolicy().Sanitize(content))
	if content == "" {
		return entity.NewValidationError("content", "content is required")
	}
	if len(content) > MaxContentLength {
		return entity.NewValidationError("content", fmt.Sprintf("content must be at most %d characters", MaxContentLength))
	}

	if err := u.comments.Create(ctx, entity.Comment{UserID: callerID, PostID: postID, Content: content}); err != nil {
		return fmt.Errorf("create comment: %w", err)
	}
	return nil
}

func (u *UseCase) List(ctx context.Context, postID int64, page, perPage int) (response.CommentList, error) {
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

	count, err := u.comments.Count(ctx, postID)
	if err != nil {
		return response.CommentList{}, fmt.Errorf("count comments: %w", err)
	}

	comments, err := u.comments.List(ctx, postID, perPage, offset)
	if err != nil {
		return response.CommentList{}, fmt.Errorf("list comments: %w", err)
	}

	items := make([]response.CommentItem, len(comments))
	for i, c := range comments {
		items[i] = response.CommentItem{
			CommentID: c.ID,
			PostID:    c.PostID,
			UserID:    c.UserID,
			Username:  c.Username,
			Content:   c.Content,
			CreatedAt: c.CreatedAt,
		}
	}

	return response.CommentList{Count: count, Items: items}, nil
}

func (u *UseCase) Delete(ctx context.Context, callerID, commentID int64) error {
	ownership, err := u.comments.GetForDelete(ctx, commentID)
	if err != nil {
		return fmt.Errorf("get comment for delete: %w", err)
	}
	if ownership.AuthorID != callerID && ownership.PostOwnerID != callerID {
		return entity.ErrForbidden
	}

	if err := u.comments.SoftDelete(ctx, commentID, ownership.PostID); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}
