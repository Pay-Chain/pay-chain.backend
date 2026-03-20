package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"payment-kita.backend/internal/domain/entities"
)

type paymentQuoteExpiryRepoStub struct {
	expired    []*entities.PaymentQuote
	fetchErr   error
	expireErr  error
	expiredIDs []uuid.UUID
}

func (s *paymentQuoteExpiryRepoStub) GetExpiredActive(_ context.Context, _ int) ([]*entities.PaymentQuote, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return s.expired, nil
}

func (s *paymentQuoteExpiryRepoStub) ExpireQuotes(_ context.Context, ids []uuid.UUID) error {
	s.expiredIDs = append([]uuid.UUID(nil), ids...)
	return s.expireErr
}

type partnerSessionExpiryRepoStub struct {
	expired    []*entities.PartnerPaymentSession
	fetchErr   error
	expireErr  error
	expiredIDs []uuid.UUID
}

func (s *partnerSessionExpiryRepoStub) GetExpiredPending(_ context.Context, _ int) ([]*entities.PartnerPaymentSession, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return s.expired, nil
}

func (s *partnerSessionExpiryRepoStub) ExpireSessions(_ context.Context, ids []uuid.UUID) error {
	s.expiredIDs = append([]uuid.UUID(nil), ids...)
	return s.expireErr
}

func TestPaymentQuoteExpiryJob_ProcessExpiredQuotes(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &paymentQuoteExpiryRepoStub{expired: []*entities.PaymentQuote{{ID: id1}, {ID: id2}}}
	job := &PaymentQuoteExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
	job.processExpiredQuotes(context.Background())
	require.Equal(t, []uuid.UUID{id1, id2}, repo.expiredIDs)
}

func TestPaymentQuoteExpiryJob_ErrorBranches(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		repo := &paymentQuoteExpiryRepoStub{fetchErr: errors.New("fetch failed")}
		job := &PaymentQuoteExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
		job.processExpiredQuotes(context.Background())
		require.Nil(t, repo.expiredIDs)
	})

	t.Run("expire error", func(t *testing.T) {
		id := uuid.New()
		repo := &paymentQuoteExpiryRepoStub{
			expired:   []*entities.PaymentQuote{{ID: id}},
			expireErr: errors.New("expire failed"),
		}
		job := &PaymentQuoteExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
		job.processExpiredQuotes(context.Background())
		require.Equal(t, []uuid.UUID{id}, repo.expiredIDs)
	})
}

func TestPartnerPaymentSessionExpiryJob_ProcessExpiredSessions(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &partnerSessionExpiryRepoStub{expired: []*entities.PartnerPaymentSession{{ID: id1}, {ID: id2}}}
	job := &PartnerPaymentSessionExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
	job.processExpiredSessions(context.Background())
	require.Equal(t, []uuid.UUID{id1, id2}, repo.expiredIDs)
}

func TestPartnerPaymentSessionExpiryJob_ErrorBranches(t *testing.T) {
	t.Run("fetch error", func(t *testing.T) {
		repo := &partnerSessionExpiryRepoStub{fetchErr: errors.New("fetch failed")}
		job := &PartnerPaymentSessionExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
		job.processExpiredSessions(context.Background())
		require.Nil(t, repo.expiredIDs)
	})

	t.Run("expire error", func(t *testing.T) {
		id := uuid.New()
		repo := &partnerSessionExpiryRepoStub{
			expired:   []*entities.PartnerPaymentSession{{ID: id}},
			expireErr: errors.New("expire failed"),
		}
		job := &PartnerPaymentSessionExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}
		job.processExpiredSessions(context.Background())
		require.Equal(t, []uuid.UUID{id}, repo.expiredIDs)
	})
}
