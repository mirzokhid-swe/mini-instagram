package entity

type User struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	Bio          string `json:"bio"`
	PasswordHash string `json:"-"`
	IsActive     bool   `json:"is_active"`
	AvatarPath   string `json:"avatar_path"`
}
