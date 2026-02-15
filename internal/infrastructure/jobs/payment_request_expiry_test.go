package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"pay-chain.backend/internal/domain/entities"
)

type paymentRequestExpiryRepoStub struct {
	expired    []*entities.PaymentRequest
	getErr     error
	expireErr  error
	expireCall int
	lastIDs    []uuid.UUID
}

func (s *paymentRequestExpiryRepoStub) GetExpiredPending(_ context.Context, _ int) ([]*entities.PaymentRequest, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.expired, nil
}

func (s *paymentRequestExpiryRepoStub) ExpireRequests(_ context.Context, ids []uuid.UUID) error {
	s.expireCall++
	s.lastIDs = ids
	return s.expireErr
}

func TestProcessExpiredRequests_NoItems(t *testing.T) {
	repo := &paymentRequestExpiryRepoStub{expired: []*entities.PaymentRequest{}}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	job.processExpiredRequests(context.Background())
	require.Equal(t, 0, repo.expireCall)
}

func TestProcessExpiredRequests_Success(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	repo := &paymentRequestExpiryRepoStub{expired: []*entities.PaymentRequest{{ID: id1}, {ID: id2}}}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	job.processExpiredRequests(context.Background())
	require.Equal(t, 1, repo.expireCall)
	require.ElementsMatch(t, []uuid.UUID{id1, id2}, repo.lastIDs)
}

func TestProcessExpiredRequests_GetError(t *testing.T) {
	repo := &paymentRequestExpiryRepoStub{getErr: errors.New("db down")}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	job.processExpiredRequests(context.Background())
	require.Equal(t, 0, repo.expireCall)
}

func TestProcessExpiredRequests_ExpireError(t *testing.T) {
	id := uuid.New()
	repo := &paymentRequestExpiryRepoStub{expired: []*entities.PaymentRequest{{ID: id}}, expireErr: errors.New("update failed")}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	job.processExpiredRequests(context.Background())
	require.Equal(t, 1, repo.expireCall)
	require.Equal(t, []uuid.UUID{id}, repo.lastIDs)
}

func TestStartStop_StopsByContext(t *testing.T) {
	repo := &paymentRequestExpiryRepoStub{expired: []*entities.PaymentRequest{}}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		job.Start(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("job did not stop on context cancel")
	}
}

func TestStartStop_StopsByStopChannel(t *testing.T) {
	repo := &paymentRequestExpiryRepoStub{expired: []*entities.PaymentRequest{}}
	job := &PaymentRequestExpiryJob{repo: repo, interval: time.Millisecond, stop: make(chan struct{})}

	done := make(chan struct{})
	go func() {
		job.Start(context.Background())
		close(done)
	}()
	job.Stop()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("job did not stop on Stop()")
	}
}
