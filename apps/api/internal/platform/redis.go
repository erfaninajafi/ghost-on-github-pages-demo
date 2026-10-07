package platform

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type Redis struct{ Client *redis.Client }

func (r Redis) Get(ctx context.Context, key string) ([]byte, error) {
	return r.Client.Get(ctx, key).Bytes()
}
func (r Redis) Set(ctx context.Context, key string, b []byte, ttl time.Duration) error {
	return r.Client.Set(ctx, key, b, ttl).Err()
}
func (r Redis) Revision(ctx context.Context) (string, error) {
	v, e := r.Client.Get(ctx, "blog:revision").Result()
	if e == redis.Nil {
		return "0", nil
	}
	return v, e
}
func (r Redis) Invalidate(ctx context.Context) error {
	return r.Client.Incr(ctx, "blog:revision").Err()
}
func (r Redis) Rank(ctx context.Context, id string, count int64) error {
	return r.Client.ZAddArgs(ctx, "blog:ranking", redis.ZAddArgs{GT: true, Members: []redis.Z{{Score: float64(count), Member: id}}}).Err()
}

// Redis-side increment and expiry are atomic across all API replicas.
var limiter = redis.NewScript(`local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('EXPIRE',KEYS[1],ARGV[1]) end; return n`)

func (r Redis) Allow(ctx context.Context, key string, max int64) (bool, error) {
	n, e := limiter.Run(ctx, r.Client, []string{"limit:" + key}, 60).Int64()
	return n <= max, e
}
