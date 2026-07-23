package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"mini-instagram/internal/entity"
	"mini-instagram/pkg/postgres"
)

type UserRepo struct {
	pool *postgres.Postgres
}

func NewUserRepo(pg *postgres.Postgres) *UserRepo {
	return &UserRepo{pool: pg}
}

func (r *UserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var exists bool
	if err := r.pool.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email uniqueness: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	if err := r.pool.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username).Scan(&exists); err != nil {
		return false, fmt.Errorf("check username uniqueness: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (entity.User, error) {
	const query = `
		SELECT id, username, email, full_name, bio, avatar_path, password, is_active
		FROM users
		WHERE email = $1
		LIMIT 1`

	var user entity.User
	err := r.pool.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.Bio,
		&user.AvatarPath,
		&user.PasswordHash,
		&user.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.User{}, entity.ErrNotFound
	}
	if err != nil {
		return entity.User{}, fmt.Errorf("find user by email: %w", err)
	}

	return user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id int64) (entity.User, error) {
	const query = `
		SELECT id, username, email, full_name, bio, avatar_path, password, is_active
		FROM users
		WHERE id = $1
		LIMIT 1`

	var user entity.User
	err := r.pool.Pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.Bio,
		&user.AvatarPath,
		&user.PasswordHash,
		&user.IsActive,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.User{}, entity.ErrNotFound
	}
	if err != nil {
		return entity.User{}, fmt.Errorf("find user by id: %w", err)
	}

	return user, nil
}

func (r *UserRepo) GetProfileStats(ctx context.Context, userID int64) (postsCount, followersCount, followingCount int64, err error) {
	const query = `
		SELECT
			(SELECT COUNT(*) FROM posts WHERE user_id = $1 AND deleted_at IS NULL),
			(SELECT COUNT(*) FROM follows WHERE following_id = $1),
			(SELECT COUNT(*) FROM follows WHERE follower_id = $1)`

	if err := r.pool.Pool.QueryRow(ctx, query, userID).Scan(&postsCount, &followersCount, &followingCount); err != nil {
		return 0, 0, 0, fmt.Errorf("get profile stats: %w", err)
	}
	return postsCount, followersCount, followingCount, nil
}

func (r *UserRepo) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	var exists bool
	if err := r.pool.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)`,
		followerID, followingID,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check following: %w", err)
	}
	return exists, nil
}

func (r *UserRepo) Create(ctx context.Context, user entity.User) (entity.User, error) {
	const query = `
		INSERT INTO users (email, full_name, username, bio, avatar_path, password, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id, username, email, full_name, bio, avatar_path, is_active`

	err := r.pool.Pool.QueryRow(ctx, query,
		user.Email,
		user.FullName,
		user.Username,
		user.Bio,
		user.AvatarPath,
		user.PasswordHash,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.Bio,
		&user.AvatarPath,
		&user.IsActive,
	)
	if err == nil {
		return user, nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		switch pgErr.ConstraintName {
		case "users_email_key":
			return entity.User{}, entity.ErrEmailTaken
		case "users_username_key":
			return entity.User{}, entity.ErrUsernameTaken
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return entity.User{}, entity.ErrNotFound
	}

	return entity.User{}, fmt.Errorf("create user: %w", err)
}
