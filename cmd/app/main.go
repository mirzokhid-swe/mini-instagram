package main

import (
	"fmt"
	"os"

	"mini-instagram/config"
	"mini-instagram/internal/app"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}

	app.Run(cfg)
}
