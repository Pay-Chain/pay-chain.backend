package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"pay-chain.backend/internal/config"
	"pay-chain.backend/internal/domain/entities"
	domainrepo "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/usecases"
)

func TestParseUserID(t *testing.T) {
	if _, err := parseUserID(""); err == nil {
		t.Fatal("expected error for empty user id")
	}
	if _, err := parseUserID("bad-uuid"); err == nil {
		t.Fatal("expected error for invalid uuid")
	}

	id := uuid.New()
	got, err := parseUserID(id.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Fatalf("expected %s got %s", id, got)
	}
}

func TestResolveAPIKeyName(t *testing.T) {
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	if got := resolveAPIKeyName("custom", now); got != "custom" {
		t.Fatalf("expected custom got %s", got)
	}
	if got := resolveAPIKeyName("", now); got != "frontend-proxy-admin-20260215-120000" {
		t.Fatalf("unexpected generated name: %s", got)
	}
}

func TestMain_ExitsWhenUserIDMissing(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_ADMIN_APIKEY") == "1" {
		os.Args = []string{"admin-apikey"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ExitsWhenUserIDMissing")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_ADMIN_APIKEY=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected helper process to fail when --user-id is missing")
	}
}

func TestMain_ExitsOnDBConnectionFailure(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_ADMIN_APIKEY") == "2" {
		os.Args = []string{"admin-apikey", "-user-id", os.Getenv("HELPER_USER_ID")}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ExitsOnDBConnectionFailure")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_ADMIN_APIKEY=2",
		"HELPER_USER_ID="+uuid.NewString(),
		"DB_HOST=127.0.0.1",
		"DB_PORT=1",
		"DB_USER=postgres",
		"DB_PASSWORD=postgres",
		"DB_NAME=paychain",
		"DB_SSLMODE=disable",
	)
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected helper process to fail on DB connection")
	}
}

func TestMain_ExitsOnInvalidUserIDFormat(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_ADMIN_APIKEY") == "3" {
		os.Args = []string{"admin-apikey", "-user-id", "invalid-uuid"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMain_ExitsOnInvalidUserIDFormat")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_ADMIN_APIKEY=3")
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected helper process to fail on invalid user-id format")
	}
}

type fakeAdminRuntime struct {
	user      *entities.User
	getErr    error
	createErr error
	resp      *entities.CreateApiKeyResponse
}

func (f fakeAdminRuntime) GetUserByID(context.Context, uuid.UUID) (*entities.User, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.user, nil
}

func (f fakeAdminRuntime) CreateApiKey(context.Context, uuid.UUID, *entities.CreateApiKeyInput) (*entities.CreateApiKeyResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return f.resp, nil
}

func TestRunAdminAPIKey_Branches(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)
	cfg := &config.Config{}

	t.Run("flag parse error", func(t *testing.T) {
		err := runAdminAPIKey([]string{"-unknown-flag"}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{}, nopCloser{}, nil
			},
			now: nowFunc(now),
		})
		if err == nil {
			t.Fatal("expected parse error")
		}
	})

	t.Run("prepare error", func(t *testing.T) {
		err := runAdminAPIKey([]string{"-user-id", userID.String()}, adminAPIKeyDeps{
			loadEnv: func() error { return errors.New("no env") },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return nil, nil, errors.New("db failed")
			},
			now: nowFunc(now),
		})
		if err == nil || !strings.Contains(err.Error(), "db failed") {
			t.Fatalf("expected prepare error, got %v", err)
		}
	})

	t.Run("user load error", func(t *testing.T) {
		err := runAdminAPIKey([]string{"-user-id", userID.String()}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{getErr: errors.New("not found")}, nopCloser{}, nil
			},
			now: nowFunc(now),
		})
		if err == nil || !strings.Contains(err.Error(), "failed to load user") {
			t.Fatalf("expected load user error, got %v", err)
		}
	})

	t.Run("non admin user", func(t *testing.T) {
		err := runAdminAPIKey([]string{"-user-id", userID.String()}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{user: &entities.User{ID: userID, Role: entities.UserRoleUser}}, nopCloser{}, nil
			},
			now: nowFunc(now),
		})
		if err == nil || !strings.Contains(err.Error(), "is not ADMIN") {
			t.Fatalf("expected non-admin error, got %v", err)
		}
	})

	t.Run("create api key error", func(t *testing.T) {
		err := runAdminAPIKey([]string{"-user-id", userID.String()}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{
					user:      &entities.User{ID: userID, Role: entities.UserRoleAdmin},
					createErr: errors.New("boom"),
				}, nopCloser{}, nil
			},
			now: nowFunc(now),
		})
		if err == nil || !strings.Contains(err.Error(), "failed creating api key") {
			t.Fatalf("expected create error, got %v", err)
		}
	})

	t.Run("success output", func(t *testing.T) {
		var out bytes.Buffer
		keyID := uuid.New()
		err := runAdminAPIKey([]string{"-user-id", userID.String()}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{
					user: &entities.User{ID: userID, Role: entities.UserRoleAdmin},
					resp: &entities.CreateApiKeyResponse{
						ID:        keyID,
						Name:      "frontend-proxy-admin-20260216-100000",
						ApiKey:    "pk_live_x",
						SecretKey: "sk_live_y",
					},
				}, nopCloser{}, nil
			},
			now: nowFunc(now),
			out: &out,
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(out.String(), "Created ADMIN API key and stored in DB") {
			t.Fatalf("unexpected output: %s", out.String())
		}
		if !strings.Contains(out.String(), "API_KEY=pk_live_x") {
			t.Fatalf("missing api key in output: %s", out.String())
		}
	})

	t.Run("nil closer fallback branch", func(t *testing.T) {
		var out bytes.Buffer
		err := runAdminAPIKey([]string{"-user-id", userID.String(), "-name", "explicit-name"}, adminAPIKeyDeps{
			loadEnv: func() error { return nil },
			loadCfg: func() *config.Config { return cfg },
			prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
				return fakeAdminRuntime{
					user: &entities.User{ID: userID, Role: entities.UserRoleAdmin},
					resp: &entities.CreateApiKeyResponse{
						ID:        uuid.New(),
						Name:      "explicit-name",
						ApiKey:    "pk_explicit",
						SecretKey: "sk_explicit",
					},
				}, nil, nil
			},
			now: nowFunc(now),
			out: &out,
		})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(out.String(), "name=explicit-name") {
			t.Fatalf("unexpected output: %s", out.String())
		}
	})
}

func TestRunAdminAPIKey_UsesDefaultInjectedFunctionsWhenNil(t *testing.T) {
	userID := uuid.New()
	err := runAdminAPIKey([]string{"-user-id", userID.String(), "-name", "from-default"}, adminAPIKeyDeps{
		loadEnv: func() error { return nil },
		loadCfg: func() *config.Config {
			cfg := &config.Config{}
			cfg.Database.Host = "localhost"
			cfg.Database.Port = -1
			cfg.Database.User = "postgres"
			cfg.Database.Password = "postgres"
			cfg.Database.DBName = "paychain"
			cfg.Database.SSLMode = "disable"
			return cfg
		},
		prepare: nil,
		now:     nowFunc(time.Date(2026, 2, 16, 10, 0, 0, 0, time.UTC)),
		out:     &bytes.Buffer{},
	})
	if err == nil {
		t.Fatal("expected default prepare to fail with invalid db config")
	}
}

func TestRunAdminAPIKey_DefaultNilsForLoaders(t *testing.T) {
	userID := uuid.New()
	var out bytes.Buffer
	err := runAdminAPIKey([]string{"-user-id", userID.String(), "-name", "nil-defaults"}, adminAPIKeyDeps{
		loadEnv: nil,
		loadCfg: nil,
		now:     nil,
		prepare: func(*config.Config) (adminAPIKeyRuntime, io.Closer, error) {
			return fakeAdminRuntime{
				user: &entities.User{ID: userID, Role: entities.UserRoleAdmin},
				resp: &entities.CreateApiKeyResponse{
					ID:        uuid.New(),
					Name:      "nil-defaults",
					ApiKey:    "pk_nil",
					SecretKey: "sk_nil",
				},
			}, nopCloser{}, nil
		},
		out: &out,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(out.String(), "API_KEY=pk_nil") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func nowFunc(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

type runtimeUserRepoStub struct {
	user *entities.User
	err  error
}

func (s runtimeUserRepoStub) Create(context.Context, *entities.User) error { return nil }
func (s runtimeUserRepoStub) GetByID(context.Context, uuid.UUID) (*entities.User, error) {
	return s.user, s.err
}
func (s runtimeUserRepoStub) GetByEmail(context.Context, string) (*entities.User, error) {
	return nil, errors.New("unused")
}
func (s runtimeUserRepoStub) Update(context.Context, *entities.User) error            { return nil }
func (s runtimeUserRepoStub) UpdatePassword(context.Context, uuid.UUID, string) error { return nil }
func (s runtimeUserRepoStub) SoftDelete(context.Context, uuid.UUID) error             { return nil }
func (s runtimeUserRepoStub) List(context.Context, string) ([]*entities.User, error)  { return nil, nil }

type runtimeAPIKeyRepoStub struct {
	createErr error
}

func (s runtimeAPIKeyRepoStub) Create(context.Context, *entities.ApiKey) error { return s.createErr }
func (s runtimeAPIKeyRepoStub) FindByKeyHash(context.Context, string) (*entities.ApiKey, error) {
	return nil, errors.New("unused")
}
func (s runtimeAPIKeyRepoStub) FindByUserID(context.Context, uuid.UUID) ([]*entities.ApiKey, error) {
	return nil, errors.New("unused")
}
func (s runtimeAPIKeyRepoStub) FindByID(context.Context, uuid.UUID) (*entities.ApiKey, error) {
	return nil, errors.New("unused")
}
func (s runtimeAPIKeyRepoStub) Update(context.Context, *entities.ApiKey) error { return nil }
func (s runtimeAPIKeyRepoStub) Delete(context.Context, uuid.UUID) error        { return nil }

func TestAdminRuntimeImpl_WrapperMethods(t *testing.T) {
	userID := uuid.New()
	user := &entities.User{ID: userID, Role: entities.UserRoleAdmin}

	var userRepo domainrepo.UserRepository = runtimeUserRepoStub{user: user}
	apiKeyCase := usecases.NewApiKeyUsecase(runtimeAPIKeyRepoStub{}, userRepo, "0000000000000000000000000000000000000000000000000000000000000000")
	rt := adminAPIKeyRuntimeImpl{userRepo: userRepo, apiKeyCase: apiKeyCase}

	got, err := rt.GetUserByID(context.Background(), userID)
	if err != nil || got == nil || got.ID != userID {
		t.Fatalf("GetUserByID wrapper failed: user=%v err=%v", got, err)
	}

	resp, err := rt.CreateApiKey(context.Background(), userID, &entities.CreateApiKeyInput{
		Name:        "k1",
		Permissions: []string{"*"},
	})
	if err != nil || resp == nil || resp.ApiKey == "" || resp.SecretKey == "" {
		t.Fatalf("CreateApiKey wrapper failed: resp=%v err=%v", resp, err)
	}
}

func TestDefaultAdminAPIKeyDeps_PrepareBranch(t *testing.T) {
	deps := defaultAdminAPIKeyDeps()
	if deps.loadEnv == nil || deps.loadCfg == nil || deps.prepare == nil || deps.now == nil || deps.out == nil {
		t.Fatalf("default deps must not be nil")
	}

	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = -1
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.DBName = "paychain"
	cfg.Database.SSLMode = "disable"

	_, _, err := deps.prepare(cfg)
	if err == nil {
		t.Fatalf("expected prepare to fail with invalid db config")
	}

	origOpen := openAdminAPIKeyDB
	defer func() { openAdminAPIKeyDB = origOpen }()
	openAdminAPIKeyDB = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file:admin_apikey_prepare_success?mode=memory&cache=shared"), &gorm.Config{})
	}

	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	runtime, closer, err := deps.prepare(cfg)
	if err != nil {
		t.Fatalf("expected prepare success with mocked db, got %v", err)
	}
	if runtime == nil || closer == nil {
		t.Fatalf("expected runtime and closer, got runtime=%v closer=%v", runtime, closer)
	}
	_ = closer.Close()
}

func TestDefaultAdminAPIKeyDeps_Prepare_SQLDBInitErrorBranch(t *testing.T) {
	deps := defaultAdminAPIKeyDeps()
	cfg := &config.Config{}
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432

	origOpen := openAdminAPIKeyDB
	origOpenSQL := openAdminSQLDB
	defer func() {
		openAdminAPIKeyDB = origOpen
		openAdminSQLDB = origOpenSQL
	}()

	openAdminAPIKeyDB = func(string) (*gorm.DB, error) {
		return gorm.Open(sqlite.Open("file:admin_apikey_sql_err?mode=memory&cache=shared"), &gorm.Config{})
	}
	openAdminSQLDB = func(*gorm.DB) (io.Closer, error) {
		return nil, errors.New("sql db init failed")
	}

	_, _, err := deps.prepare(cfg)
	if err == nil || !strings.Contains(err.Error(), "failed to init sql db") {
		t.Fatalf("expected sql db init error, got %v", err)
	}
}
