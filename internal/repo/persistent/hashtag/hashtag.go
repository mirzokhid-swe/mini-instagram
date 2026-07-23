package hashtag

import (
	"context"
	"fmt"

	"mini-instagram/internal/entity"
	"mini-instagram/pkg/postgres"
)

type HashtagRepo struct {
	pool *postgres.Postgres
}

func NewHashtagRepo(pg *postgres.Postgres) *HashtagRepo {
	return &HashtagRepo{pool: pg}
}

func (r *HashtagRepo) CountByTag(ctx context.Context, tag string) (int64, error) {
	const query = `
		SELECT COUNT(*)
		FROM posts p
		JOIN post_hashtags ph ON ph.post_id = p.id
		JOIN hashtags h ON h.id = ph.hashtag_id
		WHERE h.name = $1 AND p.deleted_at IS NULL`

	var count int64
	if err := r.pool.Pool.QueryRow(ctx, query, tag).Scan(&count); err != nil {
		return 0, fmt.Errorf("count posts by hashtag: %w", err)
	}
	return count, nil
}

func (r *HashtagRepo) ListByTag(ctx context.Context, tag string, limit, offset int) ([]entity.HashtagPost, error) {
	const query = `
		SELECT p.id, p.user_id, u.username, p.caption, p.thumbnail_path, p.like_count, p.comment_count, p.created_at
		FROM posts p
		JOIN post_hashtags ph ON ph.post_id = p.id
		JOIN hashtags h ON h.id = ph.hashtag_id
		JOIN users u ON u.id = p.user_id
		WHERE h.name = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Pool.Query(ctx, query, tag, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list posts by hashtag: %w", err)
	}
	defer rows.Close()

	var posts []entity.HashtagPost
	for rows.Next() {
		var p entity.HashtagPost
		if err := rows.Scan(&p.ID, &p.UserID, &p.Username, &p.Caption, &p.ThumbnailPath, &p.LikeCount, &p.CommentCount, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan hashtag post row: %w", err)
		}
		posts = append(posts, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hashtag post rows: %w", err)
	}

	return posts, nil
}
