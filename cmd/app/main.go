package main

import (
	"fmt"
	"os"

	"mini-instagram/config"
	"mini-instagram/internal/app"
)

// @title						Mini Instagram API
// @version					1.0
// @description				REST API for a minimal Instagram-like service: auth, profiles, posts, feed, likes, comments, follows, notifications, search and hashtags.
// @host						localhost:8080
// @BasePath					/api/v1
// @securityDefinitions.apikey	BearerAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and the JWT access token, e.g. "Bearer eyJhbGciOi...".
func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}

	app.Run(cfg)
}
