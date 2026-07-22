package auth

import (
	"context"
	"errors"
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

func (u *UseCase) Login(ctx context.Context, input request.Login) (string, error) {
	email := strings.TrimSpace(strings.ToLower(input.Email))

	user, err := u.users.FindByEmail(ctx, email)
	if errors.Is(err, entity.ErrNotFound) {
		u.logger.Info("login rejected", "email", email, "reason", "invalid credentials")
		return "", entity.ErrInvalidCredentials
	}
	if err != nil {
		u.logger.Error("login user lookup failed", "email", email, "error", err)
		return "", err
	}
	if !user.IsActive {
		u.logger.Info("login rejected", "user_id", user.ID, "reason", "invalid credentials")
		return "", entity.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		u.logger.Info("login rejected", "user_id", user.ID, "reason", "invalid credentials")
		return "", entity.ErrInvalidCredentials
	}

	accessToken, err := u.tokens.GenerateAccessToken(user)
	if err != nil {
		u.logger.Error("login access token generation failed", "user_id", user.ID, "error", err)
		return "", fmt.Errorf("generate access token: %w", err)
	}

	u.logger.Info("user logged in", "user_id", user.ID, "email", user.Email)
	return accessToken, nil
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
