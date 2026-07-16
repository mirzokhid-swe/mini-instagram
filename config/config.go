package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTP     HTTP
	Postgres Postgres
	Log      Log
}

type HTTP struct {
	Port string `env:"HTTP_PORT" envDefault:"8080"`
}

type Postgres struct {
	URL     string `env:"POSTGRES_URL,required"`
	PoolMax int    `env:"POSTGRES_POOL_MAX" envDefault:"10"`
}

type Log struct {
	Level string `env:"LOG_LEVEL" envDefault:"debug"`
}

func New() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	return cfg, nil
}
