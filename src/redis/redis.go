package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	KeyPlayers = "leaderboard:players"
)

type Z = redis.Z

func NewClient(client *redis.Client) Redis {
	// TODO: move to config?
	if client == nil {
		client = redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	}

	return &redisImpl{
		client: client,
	}
}

type Redis interface {
	Close()
	ZAdd(ctx context.Context, key string, member string, score float64) error
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error)
	Del(ctx context.Context, key string) error
}

type redisImpl struct {
	client *redis.Client
}

func (r *redisImpl) Close() {
	r.client.Close()
}

func (r *redisImpl) ZAdd(ctx context.Context, key string, member string, score float64) error {
	return r.client.ZAdd(ctx, key, redis.Z{Member: member, Score: score}).Err()
}

func (r *redisImpl) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (r *redisImpl) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
