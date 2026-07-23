// Package likesync implements the T22 background worker that flushes the
// Redis-buffered like/unlike events (see internal/cache/like) into
// Postgres every minute.
package likesync

import (
	"context"
	"time"

	"mini-instagram/internal/cache/like"
	"mini-instagram/internal/repo"
	"mini-instagram/pkg/logger"
)

const flushInterval = time.Minute

type Worker struct {
	cache       *like.Cache
	posts       repo.Post
	logger      logger.Interface
	lastFlushAt int64
}

func New(cache *like.Cache, posts repo.Post, l logger.Interface) *Worker {
	return &Worker{cache: cache, posts: posts, logger: l}
}

// Run blocks, flushing on a ticker until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.flush(ctx)
		}
	}
}

func (w *Worker) flush(ctx context.Context) {
	start := time.Now().Unix()
	lastFlushAt := w.lastFlushAt

	w.flushStates(ctx)
	w.flushCounts(ctx, lastFlushAt)

	w.lastFlushAt = start
}

func (w *Worker) flushStates(ctx context.Context) {
	keys, err := w.cache.ScanStateKeys(ctx)
	if err != nil {
		w.logger.Error("likesync: scan state keys failed", "error", err)
		return
	}

	for _, key := range keys {
		userID, postID, ok := like.ParseStateKey(key)
		if !ok {
			continue
		}

		value, found, err := w.cache.GetLikeState(ctx, userID, postID)
		if err != nil {
			w.logger.Error("likesync: read state failed", "key", key, "error", err)
			continue
		}
		if !found {
			// Concurrently cancelled/flushed already; nothing to do.
			continue
		}

		var dbErr error
		switch value {
		case "1":
			dbErr = w.posts.InsertLikeRow(ctx, userID, postID)
		case "0":
			dbErr = w.posts.DeleteLikeRow(ctx, userID, postID)
		default:
			w.logger.Error("likesync: unexpected like state value", "key", key, "value", value)
			continue
		}
		if dbErr != nil {
			w.logger.Error("likesync: flush like state failed, will retry", "key", key, "error", dbErr)
			continue
		}

		if err := w.cache.DeleteLikeState(ctx, userID, postID); err != nil {
			w.logger.Error("likesync: failed to clear flushed state key", "key", key, "error", err)
		}
	}
}

func (w *Worker) flushCounts(ctx context.Context, lastFlushAt int64) {
	keys, err := w.cache.ScanCountKeys(ctx)
	if err != nil {
		w.logger.Error("likesync: scan count keys failed", "error", err)
		return
	}

	for _, key := range keys {
		postID, ok := like.ParseCountKey(key)
		if !ok {
			continue
		}

		count, updatedAt, found, err := w.cache.GetCount(ctx, postID)
		if err != nil {
			w.logger.Error("likesync: read count failed", "key", key, "error", err)
			continue
		}
		if !found || updatedAt <= lastFlushAt {
			continue
		}

		if err := w.posts.UpdateLikeCount(ctx, postID, count); err != nil {
			w.logger.Error("likesync: flush like count failed, will retry", "key", key, "error", err)
		}
	}
}
