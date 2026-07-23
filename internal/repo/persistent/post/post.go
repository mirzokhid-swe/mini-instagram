package post

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"mini-instagram/internal/entity"
	"mini-instagram/pkg/postgres"
)

type PostRepo struct {
	pool *postgres.Postgres
}

func NewPostRepo(pg *postgres.Postgres) *PostRepo {
	return &PostRepo{pool: pg}
}

func (r *PostRepo) Create(ctx context.Context, post entity.Post) (entity.Post, error) {
	const query = `
		INSERT INTO posts (user_id, image_path, thumbnail_path, caption)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, image_path, thumbnail_path, caption, like_count, comment_count, created_at, updated_at`

	err := r.pool.Pool.QueryRow(ctx, query,
		post.UserID,
		post.ImagePath,
		post.ThumbnailPath,
		post.Caption,
	).Scan(
		&post.ID,
		&post.UserID,
		&post.ImagePath,
		&post.ThumbnailPath,
		&post.Caption,
		&post.LikeCount,
		&post.CommentCount,
		&post.CreatedAt,
		&post.UpdatedAt,
	)
	if err == nil {
		return post, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return entity.Post{}, entity.ErrNotFound
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Post{}, entity.ErrNotFound
	}

	return entity.Post{}, fmt.Errorf("create post: %w", err)
}

func (r *PostRepo) CountByUser(ctx context.Context, userID int64) (int64, error) {
	const query = `SELECT COUNT(*) FROM posts WHERE user_id = $1 AND deleted_at IS NULL`

	var count int64
	if err := r.pool.Pool.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count posts by user: %w", err)
	}
	return count, nil
}

func (r *PostRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]entity.Post, error) {
	const query = `
		SELECT id, thumbnail_path, caption, created_at
		FROM posts
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list posts by user: %w", err)
	}
	defer rows.Close()

	var posts []entity.Post
	for rows.Next() {
		var p entity.Post
		if err := rows.Scan(&p.ID, &p.ThumbnailPath, &p.Caption, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan post row: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate post rows: %w", err)
	}

	return posts, nil
}
