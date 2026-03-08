package result

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

	// BUGFIX: 检查ctx是否已取消，避免在已取消的上下文中继续执行
	if ctx.Err() != nil {
		return 0, ctx.Err()
	}

	s.mu.Lock()
	if len(amounts) == 0 {
		total := s.dayTotals[day]
		s.mu.Unlock()
		return total, nil
	}

	var delta int64
	for i := 0; i < len(amounts); i++ {
		if amounts[i] <= 0 {
			s.mu.Unlock()
			return 0, errors.New("negative amount")
		}
		// BUGFIX: 检查int64溢出，防止累加后变成负数
		if delta > 0 && amounts[i] > 9223372036854775807-delta {
			s.mu.Unlock()
			return 0, errors.New("amount overflow")
		}
		delta += amounts[i]
	}

	// BUGFIX: 检查累加后的total是否溢出
	// 使用数学方法检查：如果a > 0 && b > 0 && a > MaxInt64 - b，则a + b会溢出
	const maxInt64 int64 = 9223372036854775807
	if s.dayTotals[day] > maxInt64-delta {
		s.mu.Unlock()
		return 0, errors.New("total overflow")
	}
	total := s.dayTotals[day] + delta

	s.dayTotals[day] = total
	shouldNotify := total >= riskThreshold && s.notifier != nil && !s.alerted[day]
	notifier := s.notifier
	s.mu.Unlock()

	// BUGFIX: 在锁外调用notifier，避免死锁（如果notifier回调重入AddTransactions）
	// BUGFIX: 添加幂等性检查（!s.alerted[day]），防止告警风暴
	// BUGFIX: 只有notify成功才标记为已告警，保证失败后可以重试
	if shouldNotify {
		// BUGFIX: 再次检查ctx，避免在锁释放后到调用notifier之间ctx被取消
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		time.Sleep(10 * time.Millisecond) // emulate slow network I/O
		err := notifier.NotifyHighRisk(ctx, day, total)
		if err == nil {
			// BUGFIX: 只有notify成功才标记为已告警，保证账务与告警状态的一致性
			s.mu.Lock()
			s.alerted[day] = true
			s.mu.Unlock()
		}
		// BUGFIX: 如果notify失败，不标记为已告警，允许后续重试
		if err != nil {
			return total, err
		}
	}

	return total, nil
}
