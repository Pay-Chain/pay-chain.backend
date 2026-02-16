package redis

import (
	"context"
	"errors"
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

	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.NotNil(t, store)
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

	_, err = store.decrypt("zz-not-hex")
	assert.Error(t, err)
}

func TestSessionStoreEncryptDecrypt_InvalidKeyMaterial(t *testing.T) {
	store := &SessionStore{encryptionKey: []byte("short-key")}
	_, err := store.encrypt([]byte("x"))
	assert.Error(t, err)

	_, err = store.decrypt("00")
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

func TestSessionStore_GetSessionInvalidJSONPayload(t *testing.T) {
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

	// Encrypt non-JSON plaintext to force json.Unmarshal branch failure in GetSession.
	enc, err := store.encrypt([]byte("plain-text"))
	assert.NoError(t, err)

	ctx := context.Background()
	err = Set(ctx, "session:sid-bad-json", enc, time.Minute)
	assert.NoError(t, err)

	_, err = store.GetSession(ctx, "sid-bad-json")
	assert.Error(t, err)
}

func TestSessionStore_OperationHooks(t *testing.T) {
	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)

	origSet := setSessionValue
	origGet := getSessionValue
	origDel := delSessionValue
	t.Cleanup(func() {
		setSessionValue = origSet
		getSessionValue = origGet
		delSessionValue = origDel
	})

	setSessionValue = func(_ context.Context, _ string, _ interface{}, _ time.Duration) error {
		return errors.New("set failed")
	}
	err = store.CreateSession(context.Background(), "sid-hook", &SessionData{AccessToken: "a", RefreshToken: "r"}, time.Minute)
	assert.Error(t, err)

	// Successful create path through hook.
	setSessionValue = func(_ context.Context, _ string, _ interface{}, _ time.Duration) error { return nil }
	err = store.CreateSession(context.Background(), "sid-hook", &SessionData{AccessToken: "a", RefreshToken: "r"}, time.Minute)
	assert.NoError(t, err)

	getSessionValue = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("not found")
	}
	_, err = store.GetSession(context.Background(), "sid-hook")
	assert.Error(t, err)

	enc, err := store.encrypt([]byte(`{"accessToken":"ok","refreshToken":"ok2"}`))
	assert.NoError(t, err)
	getSessionValue = func(_ context.Context, _ string) (string, error) {
		return enc, nil
	}
	data, err := store.GetSession(context.Background(), "sid-hook")
	assert.NoError(t, err)
	assert.Equal(t, "ok", data.AccessToken)
	assert.Equal(t, "ok2", data.RefreshToken)

	delSessionValue = func(_ context.Context, _ string) error { return errors.New("delete failed") }
	err = store.DeleteSession(context.Background(), "sid-hook")
	assert.Error(t, err)

	delSessionValue = func(_ context.Context, _ string) error { return nil }
	err = store.DeleteSession(context.Background(), "sid-hook")
	assert.NoError(t, err)
}

func TestSessionStore_CreateSession_MarshalErrorBranch(t *testing.T) {
	store, err := NewSessionStore("0000000000000000000000000000000000000000000000000000000000000000")
	assert.NoError(t, err)

	origMarshal := marshalSessionJSON
	t.Cleanup(func() { marshalSessionJSON = origMarshal })

	marshalSessionJSON = func(v interface{}) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	err = store.CreateSession(context.Background(), "sid-marshal", &SessionData{AccessToken: "a", RefreshToken: "r"}, time.Minute)
	assert.Error(t, err)
}
