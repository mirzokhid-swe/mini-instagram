package auth

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"mini-instagram/internal/controller/restapi/v1/request"
	"mini-instagram/internal/entity"
	"mini-instagram/internal/repo"
	jwtmanager "mini-instagram/pkg/jwt"
	"mini-instagram/pkg/logger"
)

type UseCase struct {
	users  repo.User
	tokens *jwtmanager.TokenManager
	logger logger.Interface
}

func New(users repo.User, tokens *jwtmanager.TokenManager, logger logger.Interface) *UseCase {
	return &UseCase{users: users, tokens: tokens, logger: logger}
}

func (u *UseCase) CheckSignUpAvailability(ctx context.Context, email, username string) error {
	emailExists, err := u.users.EmailExists(ctx, email)
	if err != nil {
		return err
	}
	if emailExists {
		return entity.ErrEmailTaken
	}

	usernameExists, err := u.users.UsernameExists(ctx, username)
	if err != nil {
		return err
	}
	if usernameExists {
		return entity.ErrUsernameTaken
	}

	return nil
}

func (u *UseCase) SignUp(ctx context.Context, input request.SignUp) (string, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	input.FullName = strings.TrimSpace(input.FullName)
	input.Username = strings.TrimSpace(strings.ToLower(input.Username))
	input.Bio = strings.TrimSpace(input.Bio)

	if err := u.CheckSignUpAvailability(ctx, input.Email, input.Username); err != nil {
		return "", err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		u.logger.Error("signup password hash failed", "email", input.Email, "error", err)
		return "", fmt.Errorf("hash password: %w", err)
	}

	user, err := u.users.Create(ctx, entity.User{
		Email:        input.Email,
		FullName:     input.FullName,
		Username:     input.Username,
		Bio:          input.Bio,
		AvatarPath:   input.AvatarPath,
		PasswordHash: string(passwordHash),
	})
	if err != nil {
		return "", err
	}

	accessToken, err := u.tokens.GenerateAccessToken(user)
	if err != nil {
		u.logger.Error("signup access token generation failed", "user_id", user.ID, "error", err)
		return "", fmt.Errorf("generate access token: %w", err)
	}

	u.logger.Info("user signed up", "user_id", user.ID, "email", user.Email, "username", user.Username)
	return accessToken, nil
}
