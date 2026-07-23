// Package like implements the Redis-backed like/unlike write buffer (T22):
// a per-user pending state string (like:{user_id}:{post_id}) and a
// per-post counter hash (like-count:{post_id}) that the like usecase reads
// and writes directly, and that the likesync worker later flushes to
// Postgres.
package like

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"mini-instagram/pkg/redis"
)

const (
	statePrefix = "like:"
	countPrefix = "like-count:"
)

type Cache struct {
	client *redis.Client
}

func New(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func stateKey(userID, postID int64) string {
	return fmt.Sprintf("%s%d:%d", statePrefix, userID, postID)
}

func countKey(postID int64) string {
	return fmt.Sprintf("%s%d", countPrefix, postID)
}

// ParseStateKey extracts (userID, postID) from a "like:{user_id}:{post_id}"
// key, as scanned by the flush worker.
func ParseStateKey(key string) (userID, postID int64, ok bool) {
	rest, ok := strings.CutPrefix(key, statePrefix)
	if !ok {
		return 0, 0, false
	}
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	uid, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	pid, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, false
	}
	return uid, pid, true
}

// ParseCountKey extracts the postID from a "like-count:{post_id}" key.
func ParseCountKey(key string) (postID int64, ok bool) {
	rest, ok := strings.CutPrefix(key, countPrefix)
	if !ok {
		return 0, false
	}
	pid, err := strconv.ParseInt(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return pid, true
}

// GetLikeState reads the pending like state for (userID, postID). found is
// false when the key doesn't exist (no pending event), which is a normal
// cache miss, not an error.
func (c *Cache) GetLikeState(ctx context.Context, userID, postID int64) (value string, found bool, err error) {
	value, err = c.client.Get(ctx, stateKey(userID, postID)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get like state: %w", err)
	}
	return value, true, nil
}

func (c *Cache) SetLikeState(ctx context.Context, userID, postID int64, value string) error {
	if err := c.client.Set(ctx, stateKey(userID, postID), value, 0).Err(); err != nil {
		return fmt.Errorf("set like state: %w", err)
	}
	return nil
}

func (c *Cache) DeleteLikeState(ctx context.Context, userID, postID int64) error {
	if err := c.client.Del(ctx, stateKey(userID, postID)).Err(); err != nil {
		return fmt.Errorf("delete like state: %w", err)
	}
	return nil
}

// GetCount reads the cached like count and last-updated timestamp for a
// post. found is false when the hash doesn't exist yet (never initialized
// from the DB).
func (c *Cache) GetCount(ctx context.Context, postID int64) (count, updatedAt int64, found bool, err error) {
	values, err := c.client.HMGet(ctx, countKey(postID), "count", "updated_at").Result()
	if err != nil {
		return 0, 0, false, fmt.Errorf("get like count: %w", err)
	}
	if values[0] == nil {
		return 0, 0, false, nil
	}
	count = parseIntField(values, 0)
	updatedAt = parseIntField(values, 1)
	return count, updatedAt, true, nil
}

// GetCounts reads cached like counts for multiple posts via a single
// pipeline round-trip. Posts missing from the cache are omitted from the
// returned map and included in the returned missing slice.
func (c *Cache) GetCounts(ctx context.Context, postIDs []int64) (counts map[int64]int64, missing []int64, err error) {
	counts = make(map[int64]int64, len(postIDs))
	if len(postIDs) == 0 {
		return counts, nil, nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[int64]*goredis.SliceCmd, len(postIDs))
	for _, id := range postIDs {
		cmds[id] = pipe.HMGet(ctx, countKey(id), "count")
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, goredis.Nil) {
		return nil, nil, fmt.Errorf("pipeline get like counts: %w", err)
	}

	for _, id := range postIDs {
		values, err := cmds[id].Result()
		if err != nil || len(values) == 0 || values[0] == nil {
			missing = append(missing, id)
			continue
		}
		count := parseIntField(values, 0)
		counts[id] = count
	}

	return counts, missing, nil
}

// InitCount seeds the like counter hash from a known DB count, with
// updated_at = 0 so the next flush doesn't treat it as a pending change.
func (c *Cache) InitCount(ctx context.Context, postID, count int64) error {
	if err := c.client.HSet(ctx, countKey(postID), "count", count, "updated_at", 0).Err(); err != nil {
		return fmt.Errorf("init like count: %w", err)
	}
	return nil
}

// IncrCount increments the like counter by 1 and stamps updated_at with the
// current time, for the flush worker to pick up.
func (c *Cache) IncrCount(ctx context.Context, postID int64) error {
	if err := c.client.HIncrBy(ctx, countKey(postID), "count", 1).Err(); err != nil {
		return fmt.Errorf("incr like count: %w", err)
	}
	return c.touchUpdatedAt(ctx, postID)
}

// DecrCount decrements the like counter by 1, never below 0, and stamps
// updated_at with the current time.
func (c *Cache) DecrCount(ctx context.Context, postID int64) error {
	current, _, _, err := c.GetCount(ctx, postID)
	if err != nil {
		return err
	}
	if current > 0 {
		if err := c.client.HIncrBy(ctx, countKey(postID), "count", -1).Err(); err != nil {
			return fmt.Errorf("decr like count: %w", err)
		}
	}
	return c.touchUpdatedAt(ctx, postID)
}

func (c *Cache) touchUpdatedAt(ctx context.Context, postID int64) error {
	if err := c.client.HSet(ctx, countKey(postID), "updated_at", time.Now().Unix()).Err(); err != nil {
		return fmt.Errorf("touch like count updated_at: %w", err)
	}
	return nil
}

func (c *Cache) DeleteCount(ctx context.Context, postID int64) error {
	if err := c.client.Del(ctx, countKey(postID)).Err(); err != nil {
		return fmt.Errorf("delete like count: %w", err)
	}
	return nil
}

// ScanStateKeys returns every pending like state key currently buffered.
func (c *Cache) ScanStateKeys(ctx context.Context) ([]string, error) {
	return c.scanKeys(ctx, statePrefix+"*")
}

// ScanCountKeys returns every like counter key currently cached.
func (c *Cache) ScanCountKeys(ctx context.Context) ([]string, error) {
	return c.scanKeys(ctx, countPrefix+"*")
}

func (c *Cache) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		batch, next, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan %q: %w", pattern, err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func parseIntField(values []any, idx int) int64 {
	if idx >= len(values) || values[idx] == nil {
		return 0
	}
	str, ok := values[idx].(string)
	if !ok {
		return 0
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0
	}
	return v
}
