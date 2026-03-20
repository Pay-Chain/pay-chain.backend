package jobs

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"payment-kita.backend/internal/domain/entities"
)

type paymentQuoteExpiryRepo interface {
	GetExpiredActive(ctx context.Context, limit int) ([]*entities.PaymentQuote, error)
	ExpireQuotes(ctx context.Context, ids []uuid.UUID) error
}

type partnerPaymentSessionExpiryRepo interface {
	GetExpiredPending(ctx context.Context, limit int) ([]*entities.PartnerPaymentSession, error)
	ExpireSessions(ctx context.Context, ids []uuid.UUID) error
}

type PaymentQuoteExpiryJob struct {
	repo     paymentQuoteExpiryRepo
	interval time.Duration
	stop     chan struct{}
}

func NewPaymentQuoteExpiryJob(repo paymentQuoteExpiryRepo) *PaymentQuoteExpiryJob {
	return &PaymentQuoteExpiryJob{
		repo:     repo,
		interval: time.Minute,
		stop:     make(chan struct{}),
	}
}

func (j *PaymentQuoteExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-j.stop:
			return
		case <-ticker.C:
			j.processExpiredQuotes(ctx)
		}
	}
}

func (j *PaymentQuoteExpiryJob) Stop() { close(j.stop) }

func (j *PaymentQuoteExpiryJob) processExpiredQuotes(ctx context.Context) {
	expired, err := j.repo.GetExpiredActive(ctx, 100)
	if err != nil {
		log.Printf("error fetching expired payment quotes: %v", err)
		return
	}
	if len(expired) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(expired))
	for _, q := range expired {
		ids = append(ids, q.ID)
	}
	if err := j.repo.ExpireQuotes(ctx, ids); err != nil {
		log.Printf("error expiring payment quotes: %v", err)
	}
}

type PartnerPaymentSessionExpiryJob struct {
	repo     partnerPaymentSessionExpiryRepo
	interval time.Duration
	stop     chan struct{}
}

func NewPartnerPaymentSessionExpiryJob(repo partnerPaymentSessionExpiryRepo) *PartnerPaymentSessionExpiryJob {
	return &PartnerPaymentSessionExpiryJob{
		repo:     repo,
		interval: time.Minute,
		stop:     make(chan struct{}),
	}
}

func (j *PartnerPaymentSessionExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-j.stop:
			return
		case <-ticker.C:
			j.processExpiredSessions(ctx)
		}
	}
}

func (j *PartnerPaymentSessionExpiryJob) Stop() { close(j.stop) }

func (j *PartnerPaymentSessionExpiryJob) processExpiredSessions(ctx context.Context) {
	expired, err := j.repo.GetExpiredPending(ctx, 100)
	if err != nil {
		log.Printf("error fetching expired partner payment sessions: %v", err)
		return
	}
	if len(expired) == 0 {
		return
	}
	ids := make([]uuid.UUID, 0, len(expired))
	for _, s := range expired {
		ids = append(ids, s.ID)
	}
	if err := j.repo.ExpireSessions(ctx, ids); err != nil {
		log.Printf("error expiring partner payment sessions: %v", err)
	}
}
