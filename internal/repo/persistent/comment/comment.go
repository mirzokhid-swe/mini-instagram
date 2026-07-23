package comment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"mini-instagram/internal/entity"
	"mini-instagram/pkg/postgres"
)

type CommentRepo struct {
	pool *postgres.Postgres
}

func NewCommentRepo(pg *postgres.Postgres) *CommentRepo {
	return &CommentRepo{pool: pg}
}

func (r *CommentRepo) Create(ctx context.Context, comment entity.Comment) error {
	return r.pool.WithinTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, comment.PostID).Scan(&exists); err != nil {
			return fmt.Errorf("check post exists: %w", err)
		}
		if !exists {
			return entity.ErrPostNotFound
		}

		if _, err := tx.Exec(ctx,
			`INSERT INTO comments (user_id, post_id, content) VALUES ($1, $2, $3)`,
			comment.UserID, comment.PostID, comment.Content,
		); err != nil {
			return fmt.Errorf("insert comment: %w", err)
		}

		if _, err := tx.Exec(ctx, `UPDATE posts SET comment_count = comment_count + 1, updated_at = now() WHERE id = $1`, comment.PostID); err != nil {
			return fmt.Errorf("increment comment count: %w", err)
		}
		return nil
	})
}

func (r *CommentRepo) Count(ctx context.Context, postID int64) (int64, error) {
	const query = `SELECT COUNT(*) FROM comments WHERE post_id = $1 AND deleted_at IS NULL`

	var count int64
	if err := r.pool.Pool.QueryRow(ctx, query, postID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count comments: %w", err)
	}
	return count, nil
}

func (r *CommentRepo) List(ctx context.Context, postID int64, limit, offset int) ([]entity.Comment, error) {
	const query = `
		SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.post_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at ASC, c.id ASC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Pool.Query(ctx, query, postID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer rows.Close()

	var comments []entity.Comment
	for rows.Next() {
		var c entity.Comment
		if err := rows.Scan(&c.ID, &c.PostID, &c.UserID, &c.Username, &c.Content, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment row: %w", err)
		}
		comments = append(comments, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate comment rows: %w", err)
	}

	return comments, nil
}

func (r *CommentRepo) GetForDelete(ctx context.Context, commentID int64) (entity.CommentOwnership, error) {
	const query = `
		SELECT c.id, c.post_id, c.user_id, p.user_id
		FROM comments c
		JOIN posts p ON p.id = c.post_id
		WHERE c.id = $1 AND c.deleted_at IS NULL`

	var o entity.CommentOwnership
	err := r.pool.Pool.QueryRow(ctx, query, commentID).Scan(&o.CommentID, &o.PostID, &o.AuthorID, &o.PostOwnerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.CommentOwnership{}, entity.ErrCommentNotFound
	}
	if err != nil {
		return entity.CommentOwnership{}, fmt.Errorf("get comment for delete: %w", err)
	}
	return o, nil
}

func (r *CommentRepo) SoftDelete(ctx context.Context, commentID, postID int64) error {
	return r.pool.WithinTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE comments SET deleted_at = now(), updated_at = now() WHERE id = $1`, commentID); err != nil {
			return fmt.Errorf("soft delete comment: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0), updated_at = now() WHERE id = $1`, postID); err != nil {
			return fmt.Errorf("decrement comment count: %w", err)
		}
		return nil
	})
}
