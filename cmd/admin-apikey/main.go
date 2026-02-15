package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"pay-chain.backend/internal/config"
	"pay-chain.backend/internal/domain/entities"
	"pay-chain.backend/internal/infrastructure/repositories"
	"pay-chain.backend/internal/usecases"
)

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

func main() {
	userIDFlag := flag.String("user-id", "", "target user UUID (required)")
	nameFlag := flag.String("name", "", "api key display name (optional)")
	flag.Parse()

	userID, err := parseUserID(*userIDFlag)
	if err != nil {
		log.Fatal(err)
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()
	dsn := cfg.Database.URL()

	db, err := gorm.Open(postgres.New(postgres.Config{DSN: dsn, PreferSimpleProtocol: true}), &gorm.Config{PrepareStmt: false})
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to init sql db: %v", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()

	userRepo := repositories.NewUserRepository(db)
	apiKeyRepo := repositories.NewApiKeyRepository(db)
	apiKeyUsecase := usecases.NewApiKeyUsecase(apiKeyRepo, userRepo, cfg.Security.ApiKeyEncryptionKey)

	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Fatalf("failed to load user %s: %v", userID, err)
	}
	if user.Role != entities.UserRoleAdmin {
		log.Fatalf("user %s is not ADMIN (role=%s)", userID, user.Role)
	}

	name := resolveAPIKeyName(*nameFlag, time.Now())

	resp, err := apiKeyUsecase.CreateApiKey(ctx, userID, &entities.CreateApiKeyInput{
		Name:        name,
		Permissions: []string{"*"},
	})
	if err != nil {
		log.Fatalf("failed creating api key: %v", err)
	}

	fmt.Println("Created ADMIN API key and stored in DB")
	fmt.Printf("user_id=%s\n", userID.String())
	fmt.Printf("api_key_id=%s\n", resp.ID.String())
	fmt.Printf("name=%s\n", resp.Name)
	fmt.Printf("API_KEY=%s\n", resp.ApiKey)
	fmt.Printf("SECRET_KEY=%s\n", resp.SecretKey)
}
