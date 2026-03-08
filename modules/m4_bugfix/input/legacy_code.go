package input

import (
	"context"
	"errors"
	"sync"
	"time"
)

const riskThreshold int64 = 100

type RiskNotifier interface {
	NotifyHighRisk(ctx context.Context, day string, amount int64) error
}

type SettlementService struct {
	mu        sync.Mutex
	dayTotals map[string]int64
	alerted   map[string]bool
	notifier  RiskNotifier
}

func NewSettlementService(notifier RiskNotifier) *SettlementService {
	return &SettlementService{
		dayTotals: make(map[string]int64),
		alerted:   make(map[string]bool),
		notifier:  notifier,
	}
}

// AddTransactions contains intentionally buggy logic for evaluation tasks.
func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error) {
	if day == "" {
		return 0, errors.New("day is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if len(amounts) == 0 {
		return s.dayTotals[day], nil
	}

	var delta int64
	for i := 0; i < len(amounts); i++ {
		if amounts[i] <= 0 {
			return 0, errors.New("negative amount")
		}
		// BUG: no overflow check for int64 accumulation.
		delta += amounts[i]
	}

	total := s.dayTotals[day] + delta
	s.dayTotals[day] = total

	if total >= riskThreshold && s.notifier != nil {
		// BUG: notifier called under lock can deadlock on re-entrant calls.
		// BUG: not idempotent on success; repeated calls above threshold will over-notify.
		// BUG: marks alerted regardless notify error, causing failed alert to never retry.
		// BUG: ignores ctx cancellation/error from notifier.
		time.Sleep(10 * time.Millisecond) // emulate slow network I/O
		_ = s.notifier.NotifyHighRisk(ctx, day, total)
		s.alerted[day] = true
	}

	return total, nil
}
