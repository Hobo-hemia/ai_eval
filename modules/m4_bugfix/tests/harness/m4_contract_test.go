//go:build m4harness

package result

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSettlementService_AddTransactions_TableDriven(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		day            string
		amounts        []int64
		mockNotifyCall bool
		expectedTotal  int64
		expectErr      bool
	}

	cases := []testCase{
		{
			name:           "normal accumulation no notify",
			day:            "2026-03-08",
			amounts:        []int64{10, 20, 30},
			mockNotifyCall: false,
			expectedTotal:  60,
			expectErr:      false,
		},
		{
			name:           "threshold reached notify once",
			day:            "2026-03-08",
			amounts:        []int64{40, 70},
			mockNotifyCall: true,
			expectedTotal:  110,
			expectErr:      false,
		},
		{
			name:           "non-positive amount should fail",
			day:            "2026-03-08",
			amounts:        []int64{10, -1},
			mockNotifyCall: false,
			expectedTotal:  0,
			expectErr:      true,
		},
		{
			name:           "overflow should fail",
			day:            "2026-03-08",
			amounts:        []int64{math.MaxInt64, 1},
			mockNotifyCall: false,
			expectedTotal:  0,
			expectErr:      true,
		},
		{
			name:           "empty day should fail",
			day:            "",
			amounts:        []int64{10},
			mockNotifyCall: false,
			expectedTotal:  0,
			expectErr:      true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockNotifier := NewMockRiskNotifier(ctrl)
			if tc.mockNotifyCall {
				mockNotifier.EXPECT().
					NotifyHighRisk(gomock.Any(), tc.day, tc.expectedTotal).
					Return(nil).
					Times(1)
			} else {
				mockNotifier.EXPECT().
					NotifyHighRisk(gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
			}

			svc := NewSettlementService(mockNotifier)
			total, err := svc.AddTransactions(context.Background(), tc.day, tc.amounts)

			if tc.expectErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.expectedTotal, total)
		})
	}
}

func TestSettlementService_AddTransactions_NoDeadlockOnEmptyInput(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifier := NewMockRiskNotifier(ctrl)
	mockNotifier.EXPECT().
		NotifyHighRisk(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewSettlementService(mockNotifier)
	_, err := svc.AddTransactions(context.Background(), "2026-03-08", nil)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.AddTransactions(context.Background(), "2026-03-08", []int64{1})
	}()

	select {
	case <-done:
		assert.True(t, true)
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("second call blocked, potential deadlock")
	}
}

func TestSettlementService_AddTransactions_ThresholdNotifyExactlyOnce(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifier := NewMockRiskNotifier(ctrl)
	mockNotifier.EXPECT().
		NotifyHighRisk(gomock.Any(), "2026-03-08", int64(110)).
		Return(nil).
		Times(1)

	svc := NewSettlementService(mockNotifier)
	total, err := svc.AddTransactions(context.Background(), "2026-03-08", []int64{60, 50})
	assert.NoError(t, err)
	assert.Equal(t, int64(110), total)

	total, err = svc.AddTransactions(context.Background(), "2026-03-08", []int64{1})
	assert.NoError(t, err)
	assert.Equal(t, int64(111), total)
}

func TestSettlementService_AddTransactions_NotifyFailureShouldRetry(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifier := NewMockRiskNotifier(ctrl)
	gomock.InOrder(
		mockNotifier.EXPECT().
			NotifyHighRisk(gomock.Any(), "2026-03-10", int64(110)).
			Return(errors.New("temporary notify error")).
			Times(1),
		mockNotifier.EXPECT().
			NotifyHighRisk(gomock.Any(), "2026-03-10", int64(111)).
			Return(nil).
			Times(1),
	)

	svc := NewSettlementService(mockNotifier)
	total, err := svc.AddTransactions(context.Background(), "2026-03-10", []int64{60, 50})
	assert.Error(t, err)
	assert.Equal(t, int64(0), total, "failed notify should surface error and avoid false success total")

	total, err = svc.AddTransactions(context.Background(), "2026-03-10", []int64{111})
	assert.NoError(t, err)
	assert.Equal(t, int64(111), total)
}

func TestSettlementService_AddTransactions_ContextCanceledShouldRetryLater(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifier := NewMockRiskNotifier(ctrl)
	gomock.InOrder(
		mockNotifier.EXPECT().
			NotifyHighRisk(gomock.Any(), "2026-03-11", int64(100)).
			Return(context.Canceled).
			Times(1),
		mockNotifier.EXPECT().
			NotifyHighRisk(gomock.Any(), "2026-03-11", int64(100)).
			Return(nil).
			Times(1),
	)

	svc := NewSettlementService(mockNotifier)
	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	total, err := svc.AddTransactions(cancelledCtx, "2026-03-11", []int64{40, 60})
	assert.Error(t, err)
	assert.Equal(t, int64(0), total)

	total, err = svc.AddTransactions(context.Background(), "2026-03-11", []int64{40, 60})
	assert.NoError(t, err)
	assert.Equal(t, int64(100), total)
}

func TestSettlementService_AddTransactions_ReentrantNotifierNoDeadlock(t *testing.T) {
	t.Parallel()

	reentrant := &reentrantNotifier{}
	svc := NewSettlementService(reentrant)
	reentrant.svc = svc

	done := make(chan error, 1)
	go func() {
		_, err := svc.AddTransactions(context.Background(), "2026-03-08", []int64{70, 40})
		done <- err
	}()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(700 * time.Millisecond):
		t.Fatalf("detected potential deadlock: notifier re-entry did not return")
	}
}

func TestSettlementService_AddTransactions_ConcurrentTotalConsistency(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockNotifier := NewMockRiskNotifier(ctrl)
	mockNotifier.EXPECT().
		NotifyHighRisk(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	svc := NewSettlementService(mockNotifier)
	const goroutines = 64
	const perCall = int64(1)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_, err := svc.AddTransactions(context.Background(), "2026-03-09", []int64{perCall})
			errCh <- err
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		assert.NoError(t, err)
	}

	total, err := svc.AddTransactions(context.Background(), "2026-03-09", []int64{1})
	assert.NoError(t, err)
	assert.Equal(t, int64(goroutines+1), total)
}

func TestSettlementService_AddTransactions_NotifierShouldNotBlockGlobalLock(t *testing.T) {
	t.Parallel()

	notifier := &slowNotifier{
		sleep:   200 * time.Millisecond,
		started: make(chan struct{}, 1),
	}
	svc := NewSettlementService(notifier)

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = svc.AddTransactions(context.Background(), "2026-03-12", []int64{60, 50})
	}()

	select {
	case <-notifier.started:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("notifier did not start in expected time")
	}

	start := time.Now()
	_, err := svc.AddTransactions(context.Background(), "2026-03-13", []int64{1})
	elapsed := time.Since(start)
	assert.NoError(t, err)
	assert.Less(t, elapsed, 80*time.Millisecond, "slow notifier should not block unrelated day updates")

	<-done
}

type reentrantNotifier struct {
	svc *SettlementService
}

func (r *reentrantNotifier) NotifyHighRisk(ctx context.Context, day string, amount int64) error {
	if r.svc == nil {
		return errors.New("nil service")
	}
	_, err := r.svc.AddTransactions(ctx, day, []int64{1})
	return err
}

type slowNotifier struct {
	sleep   time.Duration
	started chan struct{}
}

func (s *slowNotifier) NotifyHighRisk(ctx context.Context, day string, amount int64) error {
	select {
	case s.started <- struct{}{}:
	default:
	}
	time.Sleep(s.sleep)
	return nil
}
