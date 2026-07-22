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
