package store

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type RedisClient struct {
	client *redis.Client
}

// RedisClientがDatastoreインターフェースを実装していることを保証
var _ Datastore = (*RedisClient)(nil)

func NewRedisClient(url string) (*RedisClient, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		slog.Error("Failed to parse Redis URL", slog.Any("error", err), slog.String("url", url))
		return nil, err
	}

	client := redis.NewClient(opt)
	_, err = client.Ping(ctx).Result()
	if err != nil {
		slog.Error("Failed to connect to Redis", slog.Any("error", err), slog.String("url", url))
		return nil, err
	}

	slog.Info("Successfully connected to Redis", slog.String("url", url))
	return &RedisClient{
		client: client,
	}, nil
}

func (r *RedisClient) Set(key string, value string) error {
	err := r.client.Set(ctx, key, value, 0).Err()
	if err != nil {
		slog.Error("Redis Set operation failed", slog.String("key", key), slog.Any("error", err))
	}
	return err
}

func (r *RedisClient) Get(key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err != nil && err != redis.Nil {
		slog.Error("Redis Get operation failed", slog.String("key", key), slog.Any("error", err))
	}
	return result, err
}

func (r *RedisClient) Expire(key string, duration time.Duration) error {
	err := r.client.Expire(ctx, key, duration).Err()
	if err != nil {
		slog.Error("Redis Expire operation failed", slog.String("key", key), slog.Duration("duration", duration), slog.Any("error", err))
	}
	return err
}

func (r *RedisClient) AddToList(key string, value string) error {
	err := r.client.LRem(ctx, key, 0, value).Err()
	if err != nil {
		slog.Error("Redis LRem operation failed", slog.String("key", key), slog.String("value", value), slog.Any("error", err))
		return err
	}
	err = r.client.LPush(ctx, key, value).Err()
	if err != nil {
		slog.Error("Redis LPush operation failed", slog.String("key", key), slog.String("value", value), slog.Any("error", err))
	}
	return err
}

func (r *RedisClient) GetListRange(key string, start, stop int64) ([]string, error) {
	result, err := r.client.LRange(ctx, key, start, stop).Result()
	if err != nil {
		slog.Error("Redis LRange operation failed", slog.String("key", key), slog.Int64("start", start), slog.Int64("stop", stop), slog.Any("error", err))
	}
	return result, err
}

func (r *RedisClient) GetUserPresence(uid string) (map[string]interface{}, error) {
	key := uid + "-store"
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		jst, _ := time.LoadLocation("Asia/Tokyo")
		now := time.Now().In(jst)
		return map[string]interface{}{
			"last_active_start_time": now.Format(time.RFC3339),
		}, nil
	} else if err != nil {
		slog.Error("Redis GetUserPresence operation failed", slog.String("key", key), slog.String("uid", uid), slog.Any("error", err))
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		slog.Error("Failed to unmarshal user presence data", slog.String("key", key), slog.String("uid", uid), slog.String("data", val), slog.Any("error", err))
		return nil, err
	}
	return result, nil
}

func (r *RedisClient) SetUserPresence(uid string, data map[string]interface{}) error {
	key := uid + "-store"
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("Failed to marshal user presence data", slog.String("key", key), slog.String("uid", uid), slog.Any("error", err))
		return err
	}
	err = r.client.Set(ctx, key, string(jsonData), 30*24*time.Hour).Err()
	if err != nil {
		slog.Error("Redis SetUserPresence operation failed", slog.String("key", key), slog.String("uid", uid), slog.Any("error", err))
	}
	return err
}

func (r *RedisClient) Delete(key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		slog.Error("Redis Delete operation failed", slog.String("key", key), slog.Any("error", err))
	}
	return err
}

func (r *RedisClient) RemoveFromList(key string, value string) error {
	err := r.client.LRem(ctx, key, 0, value).Err()
	if err != nil {
		slog.Error("Redis RemoveFromList operation failed", slog.String("key", key), slog.String("value", value), slog.Any("error", err))
	}
	return err
}
