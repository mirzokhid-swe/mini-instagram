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

func (r *PostRepo) Create(ctx context.Context, post entity.Post, hashtags []string) (entity.Post, error) {
	err := r.pool.WithinTx(ctx, func(tx pgx.Tx) error {
		const query = `
			INSERT INTO posts (user_id, image_path, thumbnail_path, caption)
			VALUES ($1, $2, $3, $4)
			RETURNING id, user_id, image_path, thumbnail_path, caption, like_count, comment_count, created_at, updated_at`

		err := tx.QueryRow(ctx, query,
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
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23503" {
				return entity.ErrNotFound
			}
			if errors.Is(err, pgx.ErrNoRows) {
				return entity.ErrNotFound
			}
			return fmt.Errorf("insert post: %w", err)
		}

		for _, name := range hashtags {
			var hashtagID int64
			if err := tx.QueryRow(ctx,
				`INSERT INTO hashtags (name) VALUES ($1) ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name RETURNING id`,
				name,
			).Scan(&hashtagID); err != nil {
				return fmt.Errorf("upsert hashtag %q: %w", name, err)
			}

			if _, err := tx.Exec(ctx,
				`INSERT INTO post_hashtags (post_id, hashtag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				post.ID, hashtagID,
			); err != nil {
				return fmt.Errorf("link hashtag %q: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		return entity.Post{}, err
	}
	return post, nil
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

func (r *PostRepo) Like(ctx context.Context, userID, postID int64) error {
	return r.pool.WithinTx(ctx, func(tx pgx.Tx) error {
		var ownerID int64
		err := tx.QueryRow(ctx, `SELECT user_id FROM posts WHERE id = $1 AND deleted_at IS NULL`, postID).Scan(&ownerID)
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.ErrPostNotFound
		}
		if err != nil {
			return fmt.Errorf("check post exists: %w", err)
		}

		tag, err := tx.Exec(ctx, `INSERT INTO likes (user_id, post_id) VALUES ($1, $2) ON CONFLICT (user_id, post_id) DO NOTHING`, userID, postID)
		if err != nil {
			return fmt.Errorf("insert like: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil
		}

		if _, err := tx.Exec(ctx, `UPDATE posts SET like_count = like_count + 1, updated_at = now() WHERE id = $1`, postID); err != nil {
			return fmt.Errorf("increment like count: %w", err)
		}

		if ownerID != userID {
			if _, err := tx.Exec(ctx,
				`INSERT INTO notifications (user_id, actor_id, action_type, post_id) VALUES ($1, $2, 'like', $3)`,
				ownerID, userID, postID,
			); err != nil {
				return fmt.Errorf("insert like notification: %w", err)
			}
		}
		return nil
	})
}

func (r *PostRepo) Unlike(ctx context.Context, userID, postID int64) error {
	return r.pool.WithinTx(ctx, func(tx pgx.Tx) error {
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts WHERE id = $1 AND deleted_at IS NULL)`, postID).Scan(&exists); err != nil {
			return fmt.Errorf("check post exists: %w", err)
		}
		if !exists {
			return entity.ErrPostNotFound
		}

		tag, err := tx.Exec(ctx, `DELETE FROM likes WHERE user_id = $1 AND post_id = $2`, userID, postID)
		if err != nil {
			return fmt.Errorf("delete like: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return entity.ErrNotLiked
		}

		if _, err := tx.Exec(ctx, `UPDATE posts SET like_count = GREATEST(like_count - 1, 0), updated_at = now() WHERE id = $1`, postID); err != nil {
			return fmt.Errorf("decrement like count: %w", err)
		}
		return nil
	})
}

func (r *PostRepo) GetByID(ctx context.Context, postID int64) (entity.PostDetail, error) {
	const query = `
		SELECT p.id, p.user_id, u.username, p.caption, p.image_path, p.like_count, p.comment_count, p.created_at
		FROM posts p
		JOIN users u ON u.id = p.user_id
		WHERE p.id = $1 AND p.deleted_at IS NULL`

	var p entity.PostDetail
	err := r.pool.Pool.QueryRow(ctx, query, postID).Scan(
		&p.ID, &p.UserID, &p.Username, &p.Caption, &p.ImagePath, &p.LikeCount, &p.CommentCount, &p.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.PostDetail{}, entity.ErrPostNotFound
	}
	if err != nil {
		return entity.PostDetail{}, fmt.Errorf("get post by id: %w", err)
	}
	return p, nil
}

func (r *PostRepo) IsLiked(ctx context.Context, userID, postID int64) (bool, error) {
	const query = `SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND post_id = $2)`

	var liked bool
	if err := r.pool.Pool.QueryRow(ctx, query, userID, postID).Scan(&liked); err != nil {
		return false, fmt.Errorf("check is liked: %w", err)
	}
	return liked, nil
}

func (r *PostRepo) GetForDelete(ctx context.Context, postID int64) (entity.Post, error) {
	const query = `
		SELECT id, user_id, image_path, thumbnail_path
		FROM posts
		WHERE id = $1 AND deleted_at IS NULL`

	var p entity.Post
	err := r.pool.Pool.QueryRow(ctx, query, postID).Scan(&p.ID, &p.UserID, &p.ImagePath, &p.ThumbnailPath)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.Post{}, entity.ErrPostNotFound
	}
	if err != nil {
		return entity.Post{}, fmt.Errorf("get post for delete: %w", err)
	}
	return p, nil
}

func (r *PostRepo) SoftDelete(ctx context.Context, postID int64) error {
	const query = `UPDATE posts SET deleted_at = now(), updated_at = now() WHERE id = $1`

	if _, err := r.pool.Pool.Exec(ctx, query, postID); err != nil {
		return fmt.Errorf("soft delete post: %w", err)
	}
	return nil
}

func (r *PostRepo) CountFeed(ctx context.Context, callerID int64) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM posts p
		JOIN follows f ON f.following_id = p.user_id AND f.follower_id = $1
		WHERE p.deleted_at IS NULL`

	var count int64
	if err := r.pool.Pool.QueryRow(ctx, query, callerID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count feed: %w", err)
	}
	return count, nil
}

func (r *PostRepo) ListFeed(ctx context.Context, callerID int64, limit, offset int) ([]entity.FeedPost, error) {
	const query = `
		SELECT p.id, p.user_id, u.username, p.caption, p.image_path, p.like_count, p.comment_count, p.created_at
		FROM posts p
		JOIN follows f ON f.following_id = p.user_id AND f.follower_id = $1
		JOIN users u ON u.id = p.user_id
		WHERE p.deleted_at IS NULL
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Pool.Query(ctx, query, callerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list feed: %w", err)
	}
	defer rows.Close()

	var posts []entity.FeedPost
	for rows.Next() {
		var p entity.FeedPost
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Caption, &p.ImagePath, &p.LikeCount, &p.CommentCount, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan feed row: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feed rows: %w", err)
	}

	return posts, nil
}
