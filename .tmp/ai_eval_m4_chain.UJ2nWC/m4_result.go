package result

import (
	"context"
	"errors"
	"sync"
)

const riskThreshold int64 = 100

type RiskNotifier interface {
	NotifyHighRisk(ctx context.Context, day string, amount int64) error
}

type SettlementService struct {
	mu        sync.Mutex
	dayTotals map[string]int64
	notifier  RiskNotifier
}

func NewSettlementService(notifier RiskNotifier) *SettlementService {
	return &SettlementService{
		dayTotals: make(map[string]int64),
		notifier:  notifier,
	}
}

func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error) {
	if day == "" {
		return 0, errors.New("day is required")
	}

	s.mu.Lock()
	// BUGFIX: 原代码在 len(amounts)==0 早返回路径缺少 Unlock，导致后续调用死锁；改用 defer 保证所有路径均释放锁
	defer s.mu.Unlock()

	var delta int64
	// BUGFIX: 原循环条件 i <= len(amounts) 最后一次迭代越界 panic；修正为 i < len(amounts)
	for i := 0; i < len(amounts); i++ {
		if amounts[i] < 0 {
			return 0, errors.New("negative amount")
		}
		delta += amounts[i]
	}

	total := s.dayTotals[day] + delta
	s.dayTotals[day] = total

	if total >= riskThreshold && s.notifier != nil {
		_ = s.notifier.NotifyHighRisk(ctx, day, total)
	}

	return total, nil
}
