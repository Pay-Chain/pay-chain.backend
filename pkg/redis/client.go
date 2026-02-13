package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var client *redis.Client

// Init initializes the Redis client
func Init(url, password string) error {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return err
	}

	if password != "" {
		opts.Password = password
	}

	client = redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return err
	}

	return nil
}

// SetClient sets the Redis client (used for testing)
func SetClient(c *redis.Client) {
	client = c
}

// GetClient returns the Redis client
func GetClient() *redis.Client {
	return client
}

// Set stores a key-value pair with expiration
func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func Get(ctx context.Context, key string) (string, error) {
	return client.Get(ctx, key).Result()
}

// Del removes a key
func Del(ctx context.Context, key string) error {
	return client.Del(ctx, key).Err()
}

// SetNX sets a key only if it does not exist
func SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return client.SetNX(ctx, key, value, expiration).Result()
}
