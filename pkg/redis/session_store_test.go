package redis

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestNewSessionStoreValidation(t *testing.T) {
	_, err := NewSessionStore("zz")
	assert.Error(t, err)

	_, err = NewSessionStore("0011")
	assert.Error(t, err)
}

func TestSessionStoreEncryptDecrypt(t *testing.T) {
	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)

	enc, err := store.encrypt([]byte(`{"x":1}`))
	assert.NoError(t, err)
	assert.NotEmpty(t, enc)

	dec, err := store.decrypt(enc)
	assert.NoError(t, err)
	assert.Contains(t, string(dec), `"x":1`)

	_, err = store.decrypt("00") // too short ciphertext
	assert.Error(t, err)
}

func TestSessionStoreCreateGetDeleteWithUnreachableRedis(t *testing.T) {
	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Depends on global client from other test (unreachable), should return error.
	err = store.CreateSession(ctx, "sid-1", &SessionData{AccessToken: "a", RefreshToken: "r"}, time.Minute)
	assert.Error(t, err)

	_, err = store.GetSession(ctx, "sid-1")
	assert.Error(t, err)

	err = store.DeleteSession(ctx, "sid-1")
	assert.Error(t, err)
}

func TestSessionStoreCreateGetDeleteSuccess(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Skipf("skip: miniredis unavailable in this environment: %v", err)
	}
	defer srv.Close()

	cli := goredis.NewClient(&goredis.Options{Addr: srv.Addr()})
	SetClient(cli)
	defer cli.Close()

	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)

	ctx := context.Background()
	err = store.CreateSession(ctx, "sid-ok", &SessionData{AccessToken: "a-ok", RefreshToken: "r-ok"}, time.Minute)
	assert.NoError(t, err)

	data, err := store.GetSession(ctx, "sid-ok")
	assert.NoError(t, err)
	assert.Equal(t, "a-ok", data.AccessToken)
	assert.Equal(t, "r-ok", data.RefreshToken)

	err = store.DeleteSession(ctx, "sid-ok")
	assert.NoError(t, err)

	_, err = store.GetSession(ctx, "sid-ok")
	assert.Error(t, err)
}
