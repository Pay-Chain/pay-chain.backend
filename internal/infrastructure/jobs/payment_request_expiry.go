package jobs

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"pay-chain.backend/internal/infrastructure/repositories"
)

// PaymentRequestExpiryJob handles expiring payment requests
type PaymentRequestExpiryJob struct {
	repo     *repositories.PaymentRequestRepositoryImpl
	interval time.Duration
	stop     chan struct{}
}

func NewPaymentRequestExpiryJob(repo *repositories.PaymentRequestRepositoryImpl) *PaymentRequestExpiryJob {
	return &PaymentRequestExpiryJob{
		repo:     repo,
		interval: 30 * time.Second, // Check every 30 seconds
		stop:     make(chan struct{}),
	}
}

func (j *PaymentRequestExpiryJob) Start(ctx context.Context) {
	log.Println("🕐 Starting payment request expiry job...")

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️ Payment request expiry job stopped (context cancelled)")
			return
		case <-j.stop:
			log.Println("⏹️ Payment request expiry job stopped")
			return
		case <-ticker.C:
			j.processExpiredRequests(ctx)
		}
	}
}

func (j *PaymentRequestExpiryJob) Stop() {
	close(j.stop)
}

func (j *PaymentRequestExpiryJob) processExpiredRequests(ctx context.Context) {
	// Get pending requests that have expired
	expired, err := j.repo.GetExpiredPending(ctx, 100)
	if err != nil {
		log.Printf("❌ Error fetching expired payment requests: %v", err)
		return
	}

	if len(expired) == 0 {
		return
	}

	log.Printf("🔄 Processing %d expired payment requests...", len(expired))

	// Collect IDs
	var ids []uuid.UUID
	for _, req := range expired {
		ids = append(ids, req.ID)
	}

	// Mark as expired
	if err := j.repo.ExpireRequests(ctx, ids); err != nil {
		log.Printf("❌ Error expiring payment requests: %v", err)
		return
	}

	log.Printf("✅ Expired %d payment requests", len(expired))
}
