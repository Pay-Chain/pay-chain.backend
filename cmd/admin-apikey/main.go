package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"pay-chain.backend/internal/config"
	"pay-chain.backend/internal/domain/entities"
	domainrepo "pay-chain.backend/internal/domain/repositories"
	"pay-chain.backend/internal/infrastructure/repositories"
	"pay-chain.backend/internal/usecases"
)

var openAdminAPIKeyDB = func(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{PrepareStmt: false})
}

var openAdminSQLDB = func(db *gorm.DB) (io.Closer, error) {
	return db.DB()
}

type adminAPIKeyRuntime interface {
	GetUserByID(ctx context.Context, userID uuid.UUID) (*entities.User, error)
	CreateApiKey(ctx context.Context, userID uuid.UUID, input *entities.CreateApiKeyInput) (*entities.CreateApiKeyResponse, error)
}

type adminAPIKeyDeps struct {
	loadEnv  func() error
	loadCfg  func() *config.Config
	prepare  func(cfg *config.Config) (adminAPIKeyRuntime, io.Closer, error)
	now      func() time.Time
	out      io.Writer
}

type adminAPIKeyRuntimeImpl struct {
	userRepo   domainrepo.UserRepository
	apiKeyCase *usecases.ApiKeyUsecase
}

func (r adminAPIKeyRuntimeImpl) GetUserByID(ctx context.Context, userID uuid.UUID) (*entities.User, error) {
	return r.userRepo.GetByID(ctx, userID)
}

func (r adminAPIKeyRuntimeImpl) CreateApiKey(ctx context.Context, userID uuid.UUID, input *entities.CreateApiKeyInput) (*entities.CreateApiKeyResponse, error) {
	return r.apiKeyCase.CreateApiKey(ctx, userID, input)
}

type nopCloser struct{}

func (nopCloser) Close() error { return nil }

func defaultAdminAPIKeyDeps() adminAPIKeyDeps {
	return adminAPIKeyDeps{
		loadEnv: func() error { return godotenv.Load() },
		loadCfg: config.Load,
		prepare: func(cfg *config.Config) (adminAPIKeyRuntime, io.Closer, error) {
			dsn := cfg.Database.URL()
			db, err := openAdminAPIKeyDB(dsn)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to connect db: %w", err)
			}

			sqlDB, err := openAdminSQLDB(db)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to init sql db: %w", err)
			}

			userRepo := repositories.NewUserRepository(db)
			apiKeyRepo := repositories.NewApiKeyRepository(db)
			apiKeyUsecase := usecases.NewApiKeyUsecase(apiKeyRepo, userRepo, cfg.Security.ApiKeyEncryptionKey)
			return adminAPIKeyRuntimeImpl{
				userRepo:   userRepo,
				apiKeyCase: apiKeyUsecase,
			}, sqlDB, nil
		},
		now: time.Now,
		out: os.Stdout,
	}
}

func parseUserID(userID string) (uuid.UUID, error) {
	if userID == "" {
		return uuid.Nil, fmt.Errorf("--user-id is required")
	}
	return uuid.Parse(userID)
}

func resolveAPIKeyName(input string, now time.Time) string {
	if input != "" {
		return input
	}
	return fmt.Sprintf("frontend-proxy-admin-%s", now.Format("20060102-150405"))
}

func runAdminAPIKey(args []string, deps adminAPIKeyDeps) error {
	if deps.loadEnv == nil {
		deps.loadEnv = func() error { return godotenv.Load() }
	}
	if deps.loadCfg == nil {
		deps.loadCfg = config.Load
	}
	if deps.now == nil {
		deps.now = time.Now
	}
	if deps.prepare == nil {
		def := defaultAdminAPIKeyDeps()
		deps.prepare = def.prepare
	}
	if deps.out == nil {
		deps.out = os.Stdout
	}

	fs := flag.NewFlagSet("admin-apikey", flag.ContinueOnError)
	userIDFlag := fs.String("user-id", "", "target user UUID (required)")
	nameFlag := fs.String("name", "", "api key display name (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	userID, err := parseUserID(*userIDFlag)
	if err != nil {
		return err
	}

	if err := deps.loadEnv(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := deps.loadCfg()
	runtime, closer, err := deps.prepare(cfg)
	if err != nil {
		return err
	}
	if closer == nil {
		closer = nopCloser{}
	}
	defer closer.Close()

	ctx := context.Background()
	user, err := runtime.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to load user %s: %w", userID, err)
	}
	if user.Role != entities.UserRoleAdmin {
		return fmt.Errorf("user %s is not ADMIN (role=%s)", userID, user.Role)
	}

	name := resolveAPIKeyName(*nameFlag, deps.now())

	resp, err := runtime.CreateApiKey(ctx, userID, &entities.CreateApiKeyInput{
		Name:        name,
		Permissions: []string{"*"},
	})
	if err != nil {
		return fmt.Errorf("failed creating api key: %w", err)
	}

	_, _ = fmt.Fprintln(deps.out, "Created ADMIN API key and stored in DB")
	_, _ = fmt.Fprintf(deps.out, "user_id=%s\n", userID.String())
	_, _ = fmt.Fprintf(deps.out, "api_key_id=%s\n", resp.ID.String())
	_, _ = fmt.Fprintf(deps.out, "name=%s\n", resp.Name)
	_, _ = fmt.Fprintf(deps.out, "API_KEY=%s\n", resp.ApiKey)
	_, _ = fmt.Fprintf(deps.out, "SECRET_KEY=%s\n", resp.SecretKey)
	return nil
}

func main() {
	if err := runAdminAPIKey(os.Args[1:], defaultAdminAPIKeyDeps()); err != nil {
		log.Fatal(err)
	}
}
