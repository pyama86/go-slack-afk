package store

import (
	"context"
	"encoding/json"
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
		return nil, err
	}

	client := redis.NewClient(opt)
	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, err
	}

	return &RedisClient{
		client: client,
	}, nil
}

func (r *RedisClient) Set(key string, value string) error {
	return r.client.Set(ctx, key, value, 0).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Expire(key string, duration time.Duration) error {
	return r.client.Expire(ctx, key, duration).Err()
}

func (r *RedisClient) AddToList(key string, value string) error {
	err := r.client.LRem(ctx, key, 0, value).Err()
	if err != nil {
		return err
	}
	return r.client.LPush(ctx, key, value).Err()
}

func (r *RedisClient) GetListRange(key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
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
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RedisClient) SetUserPresence(uid string, data map[string]interface{}) error {
	key := uid + "-store"
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, string(jsonData), 30*24*time.Hour).Err()
}

func (r *RedisClient) Delete(key string) error {
	return r.client.Del(ctx, key).Err()
}

func (r *RedisClient) RemoveFromList(key string, value string) error {
	return r.client.LRem(ctx, key, 0, value).Err()
}
