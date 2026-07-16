package main

import (
	"fmt"
	"os"

	"todo/config"
	"todo/internal/app"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		fmt.Println("config error:", err)
		os.Exit(1)
	}

	app.Run(cfg)
}
