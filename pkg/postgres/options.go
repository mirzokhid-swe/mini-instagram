package postgres

import "github.com/jackc/pgx/v5/pgxpool"

// Option -.
type Option func(*Postgres, *pgxpool.Config)

// MaxPoolSize -.
func MaxPoolSize(size int32) Option {
	return func(_ *Postgres, cfg *pgxpool.Config) {
		cfg.MaxConns = size
	}
}
