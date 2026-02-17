package redis

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func TestPingClient_WrapperExecutes(t *testing.T) {
	c := goredis.NewClient(&goredis.Options{Addr: "127.0.0.1:0"})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	if err := pingClient(ctx, c); err == nil {
		t.Fatal("expected ping error for invalid redis endpoint")
	}
}
