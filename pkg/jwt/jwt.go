// Package jwt provides token generation helpers.
package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"mini-instagram/internal/entity"
)

// TokenManager creates and validates JWT tokens.
type TokenManager struct {
	secret string
}

// New creates a TokenManager with the given secret.
func New(secret string) *TokenManager {
	return &TokenManager{secret: secret}
}

// Claims holds the JWT claims used throughout the service.
type Claims struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	FullName   string `json:"full_name"`
	IsActive   bool   `json:"is_active"`
	AvatarPath string `json:"avatar_path"`
	jwt.RegisteredClaims
}

// GenerateAccessToken generates a 24-hour access token for the given user.
func (m *TokenManager) GenerateAccessToken(user entity.User) (string, error) {
	claims := Claims{
		UserID:     user.ID,
		Username:   user.Username,
		Email:      user.Email,
		FullName:   user.FullName,
		IsActive:   user.IsActive,
		AvatarPath: user.AvatarPath,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

// ValidateToken parses and validates a JWT access token and returns the claims.
func (m *TokenManager) ValidateToken(tokenString string) (Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})
	if err != nil {
		return Claims{}, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return Claims{}, fmt.Errorf("invalid token claims")
	}

	return *claims, nil
}
