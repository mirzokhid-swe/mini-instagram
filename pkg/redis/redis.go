package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	*goredis.Client
}

func New(ctx context.Context, rawURL string) (*Client, error) {
	options, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse Redis URL: %w", err)
	}

	client := &Client{Client: goredis.NewClient(options)}
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping Redis: %w", err)
	}

	return client, nil
}
