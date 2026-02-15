package redis

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestInitInvalidURL(t *testing.T) {
	err := Init("://invalid-url", "")
	assert.Error(t, err)
}

func TestSetClientAndBasicOpsWithUnreachableRedis(t *testing.T) {
	cli := goredis.NewClient(&goredis.Options{
		Addr:         "127.0.0.1:0", // invalid/unreachable
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	SetClient(cli)
	assert.NotNil(t, GetClient())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	assert.Error(t, Set(ctx, "k", "v", time.Second))
	_, err := Get(ctx, "k")
	assert.Error(t, err)
	assert.Error(t, Del(ctx, "k"))
	_, err = SetNX(ctx, "k", "v", time.Second)
	assert.Error(t, err)
}
