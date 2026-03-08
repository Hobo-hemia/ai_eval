package result

import (
	"context"
	"errors"
	"math"
	"sync"
)

const riskThreshold int64 = 100

type RiskNotifier interface {
	NotifyHighRisk(ctx context.Context, day string, amount int64) error
}

type SettlementService struct {
	mu        sync.RWMutex
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

func (s *SettlementService) AddTransactions(ctx context.Context, day string, amounts []int64) (int64, error) {
	if day == "" {
		return 0, errors.New("day is required")
	}

	if len(amounts) == 0 {
		s.mu.RLock()
		total := s.dayTotals[day]
		s.mu.RUnlock()
		return total, nil
	}

	// BUGFIX: 溢出检查 - 在累加前检查是否会溢出int64
	var delta int64
	for i := 0; i < len(amounts); i++ {
		if amounts[i] <= 0 {
			return 0, errors.New("negative amount")
		}
		if delta > math.MaxInt64-amounts[i] {
			return 0, errors.New("amount overflow")
		}
		delta += amounts[i]
	}

	s.mu.Lock()
	currentTotal := s.dayTotals[day]

	// BUGFIX: 再次检查累加后是否溢出
	if currentTotal > math.MaxInt64-delta {
		s.mu.Unlock()
		return 0, errors.New("total amount overflow")
	}

	total := currentTotal + delta
	s.dayTotals[day] = total

	// BUGFIX: 检查是否已告警，避免重复告警风暴
	alreadyAlerted := s.alerted[day]
	shouldAlert := total >= riskThreshold && s.notifier != nil && !alreadyAlerted

	s.mu.Unlock()

	if shouldAlert {
		// BUGFIX: 在锁外调用notifier，避免死锁（notifier可能重入系统）
		// BUGFIX: 处理context取消和notifier错误，保证账务与告警状态一致性
		err := s.notifier.NotifyHighRisk(ctx, day, total)
		if err != nil {
			// 告警失败，不标记为已告警，允许重试
			return total, nil
		}

		// 告警成功，标记为已告警
		s.mu.Lock()
		s.alerted[day] = true
		s.mu.Unlock()
	}

	return total, nil
}
