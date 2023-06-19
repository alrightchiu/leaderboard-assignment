package redis

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

var (
	redisHostMaster  = os.Getenv("REDIS_HOST_MASTER")
	redisHostReplica = os.Getenv("REDIS_HOST_REPLICA")
	redisPortMaster  = os.Getenv("REDIS_PORT_MASTER")
	redisPortReplica = os.Getenv("REDIS_PORT_REPLICA")
)

const (
	KeyPlayers = "leaderboard:players"
)

type Z = redis.Z

// for testing
func NewMockMasterRedis(client *redis.Client) Redis {
	return &masterImpl{
		client: client,
	}
}

// for testing
func NewMockReplicaRedis(client *redis.Client) Redis {
	return &replicaImpl{
		client: client,
	}
}

func NewMasterClient() Redis {
	if redisHostMaster == "" {
		redisHostMaster = "localhost"
	}
	if redisPortMaster == "" {
		redisPortMaster = "7000"
	}

	addr := fmt.Sprintf("%s:%s", redisHostMaster, redisPortMaster)
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	s := client.Do(context.Background(), "ping").String()
	fmt.Printf("master client(%s): %s\n", addr, s)

	return &masterImpl{
		client: client,
	}
}

func NewReplicaClient() Redis {
	if redisHostReplica == "" {
		redisHostReplica = "localhost"
	}
	if redisPortReplica == "" {
		redisPortReplica = "7001"
	}

	addr := fmt.Sprintf("%s:%s", redisHostReplica, redisPortReplica)
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	s := client.Do(context.Background(), "ping").String()
	fmt.Printf("replica client(%s): %s\n", addr, s)

	return &replicaImpl{
		client: client,
	}
}

type Redis interface {
	Close() error
	ZAdd(ctx context.Context, key string, member string, score float64) error
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error)
	Del(ctx context.Context, key string) error
}

type masterImpl struct {
	client *redis.Client
}

func (r *masterImpl) Close() error {
	return r.client.Close()
}

func (r *masterImpl) ZAdd(ctx context.Context, key string, member string, score float64) error {
	return r.client.ZAdd(ctx, key, redis.Z{Member: member, Score: score}).Err()
}

func (r *masterImpl) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (r *masterImpl) Del(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

type replicaImpl struct {
	client *redis.Client
}

func (r *replicaImpl) Close() error {
	return r.client.Close()
}

func (r *replicaImpl) ZAdd(ctx context.Context, key string, member string, score float64) error {
	return errors.New("should not use ZAdd by replica")
}

func (r *replicaImpl) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRevRangeWithScores(ctx, key, start, stop).Result()
}

func (r *replicaImpl) Del(ctx context.Context, key string) error {
	return errors.New("should not use Del by replica")
}
