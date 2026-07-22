package repo

import (
	"context"

	"mini-instagram/internal/entity"
)

type User interface {
	EmailExists(ctx context.Context, email string) (bool, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	FindByEmail(ctx context.Context, email string) (entity.User, error)
	Create(ctx context.Context, user entity.User) (entity.User, error)
}
